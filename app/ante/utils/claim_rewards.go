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

package utils

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ClaimStakingRewardsIfNecessary checks if the given address has enough balance to cover the
// given amount. If not, it attempts to claim enough staking rewards to cover the amount.
// CONTRACT: the method assumes that the amount contains a single coin: staking or non-staking.
func ClaimStakingRewardsIfNecessary(
	ctx sdk.Context,
	bankKeeper BankKeeper,
	distributionKeeper DistributionKeeper,
	stakingKeeper StakingKeeper,
	addr sdk.AccAddress,
	amount sdk.Coins,
) error {
	stakingDenom := stakingKeeper.BondDenom(ctx)
	found, amountInStakingDenom := amount.Find(stakingDenom)
	if !found {
		// amount is not in staking denom, do not try to claim staking rewards
		return nil
	}

	balance := bankKeeper.GetBalance(ctx, addr, stakingDenom)
	if balance.IsNegative() {
		return errortypes.ErrInsufficientFunds.Wrapf("balance of %s in %s is negative", addr, stakingDenom)
	}

	// check if the account has enough balance to cover the fees
	if balance.IsGTE(amountInStakingDenom) {
		return nil
	}

	// Calculate the amount of staking rewards needed to cover the fees
	difference := amountInStakingDenom.Sub(balance)

	// attempt to claim enough staking rewards to cover the fees
	return ClaimSufficientStakingRewards(
		ctx, stakingKeeper, distributionKeeper, addr, difference,
	)
}

// ClaimSufficientStakingRewards checks if the account has enough staking rewards unclaimed
// to cover the given amount. If more than enough rewards are unclaimed, only those up to
// the given amount are claimed.
func ClaimSufficientStakingRewards(
	ctx sdk.Context,
	stakingKeeper StakingKeeper,
	distributionKeeper DistributionKeeper,
	addr sdk.AccAddress,
	amount sdk.Coin,
) error {
	var (
		err     error
		reward  sdk.Coins
		rewards sdk.Coins
	)

	// Allocate a cached context to avoid writing to state if there are not enough rewards
	cacheCtx, writeFn := ctx.CacheContext()

	// Iterate through delegations and get the rewards if any are unclaimed.
	// The loop stops once a sufficient amount was withdrawn.
	stakingKeeper.IterateDelegations(
		cacheCtx,
		addr,
		func(_ int64, delegation stakingtypes.DelegationI) (stop bool) {
			reward, err = distributionKeeper.WithdrawDelegationRewards(cacheCtx, addr, delegation.GetValidatorAddr())
			if err != nil {
				return true
			}
			rewards = rewards.Add(reward...)

			return rewards.AmountOf(amount.Denom).GTE(amount.Amount)
		},
	)

	// check if there was an error while iterating delegations
	if err != nil {
		return errorsmod.Wrap(err, "error while withdrawing delegation rewards")
	}

	// only write to state if there are enough rewards to cover the transaction fees
	if rewards.AmountOf(amount.Denom).LT(amount.Amount) {
		return errortypes.ErrInsufficientFee.Wrapf("insufficient staking rewards to cover transaction fees")
	}
	writeFn() // commit state changes
	return nil
}

// DeductFees deducts fees from the given account. If the account does not have enough
// SDK coins to cover the fees, it tries to cover them with respective ERC20 tokens.
func DeductFees(bankKeeper BankKeeper, erc20Keeper ERC20Keeper, ctx sdk.Context, acc types.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(errortypes.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	// first, try to pay the fee with SDK coins

	accBalances := bankKeeper.GetAllBalances(ctx, acc.GetAddress())
	if fees.IsAllLTE(accBalances) {
		// the fee can be paid with SDK coins
		err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
		if err != nil {
			return fmt.Errorf("send coins from account to module: %w", err)
		}
		return nil
	}

	// try to pay the fee with ERC20 tokens

	// use a cached context to avoid writing to state if there are not enough ERC20 balance to cover the fees
	// or some other error occurs
	cacheCtx, writeFn := ctx.CacheContext()

	// convert ERC20 token from sender's ETH address to SDK coin on sender's SDK address
	for _, fee := range fees {
		err := erc20Keeper.TryConvertErc20Sdk(cacheCtx, acc.GetAddress(), acc.GetAddress(), fee.Denom, fee.Amount)
		if err != nil {
			return fmt.Errorf("convert ERC20 token to SDK coin: denom %s: %w", fee.Denom, err)
		}
	}

	// now sender should have enough balance to cover the fees
	err := bankKeeper.SendCoinsFromAccountToModule(cacheCtx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return fmt.Errorf("send coins from account to module: %w", err)
	}

	writeFn()

	return nil
}
