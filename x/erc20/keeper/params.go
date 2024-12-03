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
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v12/x/erc20/types"
)

var isTrue = []byte("0x01")

// GetParams returns the total set of erc20 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	enableErc20 := k.IsERC20Enabled(ctx)
	enableEvmHook := k.GetEnableEVMHook(ctx)
	registrationFee := k.GetRegistrationFee(ctx)

	return types.NewParams(enableErc20, enableEvmHook, registrationFee)
}

// SetParams sets the erc20 parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	k.setERC20Enabled(ctx, params.EnableErc20)
	k.setEnableEVMHook(ctx, params.EnableEVMHook)
	err := k.setRegistrationFee(ctx, params.RegistrationFee)
	if err != nil {
		return err
	}

	return nil
}

// IsERC20Enabled returns true if the module logic is enabled
func (k Keeper) IsERC20Enabled(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.ParamStoreKeyEnableErc20)
}

// GetEnableEVMHook returns true if the EVM hooks are enabled
func (k Keeper) GetEnableEVMHook(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.ParamStoreKeyEnableEVMHook)
}

// setERC20Enabled sets the EnableERC20 param in the store
func (k Keeper) setERC20Enabled(ctx sdk.Context, enable bool) {
	store := ctx.KVStore(k.storeKey)
	if enable {
		store.Set(types.ParamStoreKeyEnableErc20, isTrue)
		return
	}
	store.Delete(types.ParamStoreKeyEnableErc20)
}

// setEnableEVMHook sets the EnableEVMHook param in the store
func (k Keeper) setEnableEVMHook(ctx sdk.Context, enable bool) {
	store := ctx.KVStore(k.storeKey)
	if enable {
		store.Set(types.ParamStoreKeyEnableEVMHook, isTrue)
		return
	}
	store.Delete(types.ParamStoreKeyEnableEVMHook)
}

// setRegistrationFee sets the RegistrationFee param in the store
func (k Keeper) setRegistrationFee(ctx sdk.Context, feeAmt math.Int) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := feeAmt.Marshal()
	if err != nil {
		return err
	}
	store.Set(types.ParamStoreKeyRegistrationFee, bz)
	return nil
}

// GetRegistrationFee returns the registration fee amount
func (k Keeper) GetRegistrationFee(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamStoreKeyRegistrationFee)
	if len(bz) == 0 {
		return types.DefaultRegistrationFee
	}

	var feeAmt math.Int
	if err := feeAmt.Unmarshal(bz); err != nil {
		k.Logger(ctx).Error("failed to unmarshal registration fee. continuing with default", "err", err)
		return types.DefaultRegistrationFee
	}
	return feeAmt
}
