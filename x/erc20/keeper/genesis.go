// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/evmos/v12/x/erc20/types"
)

// convertAccountBankToERC20 converts bank tokens to ERC20 tokens for an account using existing conversion logic
func (k Keeper) convertAccountBankToERC20(
	ctx sdk.Context,
	pair types.TokenPair,
	account sdk.AccAddress,
	balance sdk.Coin,
) error {
	receiver := common.BytesToAddress(account)

	msg := &types.MsgConvertCoin{
		Coin:     balance,
		Receiver: receiver.Hex(),
		Sender:   account.String(),
	}

	_, err := k.convertCoinNativeCoin(ctx, pair, msg, receiver, account)
	if err != nil {
		return fmt.Errorf("failed to convert bank tokens to ERC20 for account %s: %w", account.String(), err)
	}

	k.Logger(ctx).Debug(
		"auto-converted bank tokens to ERC20 on genesis",
		"account", account.String(),
		"denom", balance.Denom,
		"amount", balance.Amount.String(),
		"contract", pair.Erc20Address,
	)

	return nil
}

// autoConvertBankToERC20OnGenesis iterates through all accounts and converts their bank balances to ERC20 tokens
func (k Keeper) autoConvertBankToERC20OnGenesis(ctx sdk.Context, pair types.TokenPair) error {

	k.bankKeeper.IterateAllBalances(ctx, func(account sdk.AccAddress, coin sdk.Coin) bool {
		// Skip module accounts to avoid converting module balances
		if k.bankKeeper.BlockedAddr(account) {
			return false // continue iteration
		}

		if coin.Denom != pair.Denom {
			return false // continue iteration
		}

		// Convert bank tokens to ERC20
		if err := k.convertAccountBankToERC20(ctx, pair, account, coin); err != nil {
			k.Logger(ctx).Error(
				"failed to auto-convert bank tokens to ERC20 on genesis",
				"account", account.String(),
				"denom", coin.Denom,
				"error", err,
			)
			// Continue with other accounts even if one fails
		}

		return false // continue iteration
	})

	return nil
}

// InitGenesis import module genesis
func (k Keeper) InitGenesis(
	ctx sdk.Context,
	data types.GenesisState,
) {
	err := k.SetParams(ctx, data.Params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}

	// ensure erc20 module account is set on genesis
	if acc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		// NOTE: shouldn't occur
		panic("the erc20 module account has not been set")
	}

	for _, pair := range data.TokenPairs {
		id := pair.GetID()
		if pair.Erc20Address == types.DeployedContractOnGenesisAddr {
			metadata, found := k.bankKeeper.GetDenomMetaData(ctx, pair.Denom)
			if !found {
				panic(fmt.Errorf("metadata not found for denom %s", pair.Denom))
			}
			actualPair, err := k.RegisterCoin(ctx, metadata)
			if err != nil {
				panic(fmt.Errorf("failed to register coin: %s", err))
			}

			// Auto-convert existing bank balances to ERC20 tokens
			if err := k.autoConvertBankToERC20OnGenesis(ctx, *actualPair); err != nil {
				panic(fmt.Errorf("failed to auto-convert bank balances to ERC20 for denom %s: %s", pair.Denom, err))
			}
		} else {
			k.SetTokenPair(ctx, pair)
			k.SetDenomMap(ctx, pair.Denom, id)
			k.SetERC20Map(ctx, pair.GetERC20Contract(), id)
		}
	}
}

// ExportGenesis export module status
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		TokenPairs: k.GetTokenPairs(ctx),
	}
}
