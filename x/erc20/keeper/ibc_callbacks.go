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
	errorsmod "cosmossdk.io/errors"
	"github.com/armon/go-metrics"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	"github.com/ethereum/go-ethereum/common"

	"github.com/evmos/evmos/v12/ibc"
	"github.com/evmos/evmos/v12/x/erc20/types"
)

// OnRecvPacket performs the ICS20 middleware receive callback for automatically
// converting an IBC Coin to their ERC20 representation.
// For the conversion to succeed, the IBC denomination must have previously been
// registered. Note that the native staking denomination (e.g. "aevmos"),
// is excluded from the conversion.
//
// CONTRACT: This middleware MUST be executed transfer after the ICS20 OnRecvPacket
// Return acknowledgement and continue with the next layer of the IBC middleware
// stack if:
// - ERC20s are disabled
// - Denomination is native staking token
// Return an error acknowledgement if:
// - The base denomination is not registered as ERC20
// - The packet is received from a non-evm channel
// - The packet is received from a non-evm chain
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack exported.Acknowledgement,
) exported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// NOTE: shouldn't happen as the packet has already
		// been decoded on ICS20 transfer logic
		err = errorsmod.Wrapf(errortypes.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// use a zero gas config to avoid extra costs for the relayers
	ctx = ctx.
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})

	if !k.IsERC20Enabled(ctx) {
		return ack
	}

	// Get addresses in `evmos1` and the original bech32 format
	sender, recipient, err := ibc.GetTransferSenderRecipientFromData(data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	senderAcc := k.accountKeeper.GetAccount(ctx, sender)

	// return acknoledgement without conversion if sender is a module account
	if types.IsModuleAccount(senderAcc) {
		return ack
	}

	// parse the transferred denom
	coin := ibc.GetReceivedCoin(
		packet.SourcePort, packet.SourceChannel,
		packet.DestinationPort, packet.DestinationChannel,
		data.Denom, data.Amount,
	)

	// check if the coin is the EVM native token
	nativeDenom := k.evmKeeper.GetParams(ctx).EvmDenom
	if coin.Denom == nativeDenom {
		// no-op, received coin is the EVM nativetoken
		return ack
	}

	pairID := k.GetTokenPairID(ctx, coin.Denom)
	if len(pairID) == 0 {
		k.Logger(ctx).Error(
			"token pair not found",
			"coin", coin.Denom,
		)
		err = errorsmod.Wrapf(types.ErrTokenPairNotFound, "coin denom: %s", coin.Denom)
		return channeltypes.NewErrorAcknowledgement(err)
	}

	pair, _ := k.GetTokenPair(ctx, pairID)
	if !pair.Enabled {
		// no-op: continue with the rest of the stack without conversion
		return ack
	}

	// Instead of converting just the received coins, convert the whole user balance
	// which includes the received coins.
	balance := k.bankKeeper.GetBalance(ctx, recipient, coin.Denom)

	// Build MsgConvertCoin, from recipient to recipient since IBC transfer already occurred
	msg := types.NewMsgConvertCoin(balance, common.BytesToAddress(recipient.Bytes()), recipient)

	// NOTE: we don't use ValidateBasic the msg since we've already validated
	// the ICS20 packet data
	k.Logger(ctx).Info("I'm here")
	println("I'm also here")

	// Use MsgConvertCoin to convert the Cosmos Coin to an ERC20
	if _, err = k.ConvertCoin(sdk.WrapSDKContext(ctx), msg); err != nil {
		k.Logger(ctx).Error(
			"failed to convert coin to erc20",
			"coin", balance.String(),
			"error", err.Error(),
		)
		return channeltypes.NewErrorAcknowledgement(err)
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{types.ModuleName, "ibc", "on_recv", "total"},
			1,
			[]metrics.Label{
				telemetry.NewLabel("denom", coin.Denom),
				telemetry.NewLabel("source_channel", packet.SourceChannel),
				telemetry.NewLabel("source_port", packet.SourcePort),
			},
		)
	}()

	return ack
}

// OnAcknowledgementPacket responds to the the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement was a
// success then nothing occurs. If the acknowledgement failed, then the sender
// is refunded and then the IBC Coins are converted to ERC20.
func (k Keeper) OnAcknowledgementPacket(
	ctx sdk.Context, _ channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
	ack channeltypes.Acknowledgement,
) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		// convert the token from Cosmos Coin to its ERC20 representation
		return k.ConvertCoinToERC20FromPacket(ctx, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing needs to
		// be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutPacket converts the IBC coin to ERC20 after refunding the sender
// since the original packet sent was never received and has been timed out.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, _ channeltypes.Packet, data transfertypes.FungibleTokenPacketData) error {
	return k.ConvertCoinToERC20FromPacket(ctx, data)
}

// ConvertCoinToERC20FromPacket converts the IBC coin to ERC20 after refunding the sender
func (k Keeper) ConvertCoinToERC20FromPacket(ctx sdk.Context, data transfertypes.FungibleTokenPacketData) error {
	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}

	// use a zero gas config to avoid extra costs for the relayers
	ctx = ctx.
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})

	// assume that all module accounts on Evmos need to have their tokens in the
	// IBC representation as opposed to ERC20
	senderAcc := k.accountKeeper.GetAccount(ctx, sender)
	if types.IsModuleAccount(senderAcc) {
		return nil
	}

	coin := ibc.GetSentCoin(data.Denom, data.Amount)

	// check if the coin is a native staking token
	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if coin.Denom == bondDenom {
		// no-op, received coin is the staking denomination
		return nil
	}

	params := k.GetParams(ctx)
	if !params.EnableErc20 || !k.IsDenomRegistered(ctx, coin.Denom) {
		// no-op, ERC20s are disabled or the denom is not registered
		return nil
	}

	msg := types.NewMsgConvertCoin(coin, common.BytesToAddress(sender), sender)

	// NOTE: we don't use ValidateBasic the msg since we've already validated the
	// fields from the packet data

	// convert Coin to ERC20
	if _, err = k.ConvertCoin(sdk.WrapSDKContext(ctx), msg); err != nil {
		return err
	}

	defer func() {
		telemetry.IncrCounter(1, types.ModuleName, "ibc", "error", "total")
	}()

	return nil
}
