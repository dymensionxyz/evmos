package keeper

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/evmos/evmos/v12/contracts"
	"github.com/evmos/evmos/v12/x/erc20/types"
)

// TryConvertErc20Sdk converts ERC20 token to SDK coin from sender's ETH address to receiver's SDK address.
// If the sender do not have enough balance, the method returns error without any state changes.
// The method
//   - Burns escrowed tokens
//   - Unescrows coins that have been previously escrowed with ConvertCoin
//   - Check if coin balance increased by amount
//   - Check if token balance decreased by amount
func (k Keeper) TryConvertErc20Sdk(
	ctx sdk.Context,
	sender, receiver sdk.AccAddress,
	denom string, // denom may be either ERC20 address or SDK coin
	amount math.Int,
) error {
	pair, err := k.MintingEnabled(ctx, sender, receiver, denom)
	if err != nil {
		return fmt.Errorf("check minting is enabled: %w", err)
	}

	// NOTE: coin fields already validated
	senderEth := common.BytesToAddress(sender.Bytes())
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()

	balanceCoin := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	balanceToken := k.BalanceOf(ctx, erc20, contract, senderEth)
	if balanceToken == nil {
		return errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	if balanceToken.Cmp(amount.BigInt()) < 0 {
		return errorsmod.Wrapf(
			errortypes.ErrInsufficientFunds,
			"token balance %s < required amount %s",
			balanceToken, amount,
		)
	}

	// Burn escrowed tokens
	_, err = k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "burnCoins", senderEth, amount.BigInt())
	if err != nil {
		return err
	}

	// Unescrow coins and send to receiver
	coins := sdk.Coins{sdk.Coin{Denom: pair.Denom, Amount: amount}}
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, coins)
	if err != nil {
		return err
	}

	// Check expected receiver balance after transfer
	balanceCoinAfter := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	expCoin := balanceCoin.Add(coins[0])
	if ok := balanceCoinAfter.IsEqual(expCoin); !ok {
		return errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid coin balance - expected: %v, actual: %v",
			expCoin, balanceCoinAfter,
		)
	}

	// Check expected Sender balance after transfer
	tokens := coins[0].Amount.BigInt()
	balanceTokenAfter := k.BalanceOf(ctx, erc20, contract, senderEth)
	if balanceTokenAfter == nil {
		return errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	expToken := big.NewInt(0).Sub(balanceToken, tokens)
	if r := balanceTokenAfter.Cmp(expToken); r != 0 {
		return errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			expToken, balanceTokenAfter,
		)
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC20,
				sdk.NewAttribute(sdk.AttributeKeySender, sender.String()),
				sdk.NewAttribute(types.AttributeKeyReceiver, receiver.String()),
				sdk.NewAttribute(sdk.AttributeKeyAmount, amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		},
	)

	return nil
}
