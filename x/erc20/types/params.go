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

package types

import (
	fmt "fmt"

	"cosmossdk.io/math"
)

// Parameter store key
var (
	ParamStoreKeyEnableErc20     = []byte("EnableErc20")
	ParamStoreKeyEnableEVMHook   = []byte("EnableEVMHook")
	ParamStoreKeyRegistrationFee = []byte("RegistrationFee")

	DefaultRegistrationFee = math.NewInt(10).MulRaw(1e18) // 10 tokens of native denom
)

// NewParams creates a new Params object
func NewParams(
	enableErc20, enableEVMHook bool,
	registrationFee math.Int,
) Params {
	return Params{
		EnableErc20:     enableErc20,
		EnableEVMHook:   enableEVMHook,
		RegistrationFee: registrationFee,
	}
}

func DefaultParams() Params {
	return Params{
		EnableErc20:     true,
		EnableEVMHook:   true,
		RegistrationFee: DefaultRegistrationFee,
	}
}

func ValidateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func (p Params) Validate() error {
	if err := ValidateBool(p.EnableEVMHook); err != nil {
		return err
	}

	if err := ValidateBool(p.EnableErc20); err != nil {
		return err
	}

	if p.RegistrationFee.IsNil() || p.RegistrationFee.IsNegative() {
		return fmt.Errorf("registration fee cannot be negative: %s", p.RegistrationFee)
	}

	return nil
}
