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
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/evmos/evmos/v12/utils"
)

var (
	KeyClaimsDenom        = []byte("ClaimsDenom")
	KeyDurationUntilDecay = []byte("DurationUntilDecay")
	KeyDurationOfDecay    = []byte("DurationOfDecay")
	KeyAuthorizedChannels = []byte("AuthorizedChannels")
	KeyEVMChannels        = []byte("EVMChannels")
	KeyEnableClaims       = []byte("EnableClaims")
	KeyAirdropStartTime   = []byte("AirdropStartTime")

	// DefaultClaimsDenom is aevmos
	DefaultClaimsDenom = utils.BaseDenom
	// DefaultDurationUntilDecay is 1 month = 30.4375 days
	DefaultDurationUntilDecay = 2629800 * time.Second
	// DefaultDurationOfDecay is 2 months
	DefaultDurationOfDecay = 2 * DefaultDurationUntilDecay
	// DefaultAuthorizedChannels defines the list of default IBC authorized channels that can perform
	// IBC address attestations in order to migrate claimable amounts. By default
	// only Osmosis and Cosmos Hub channels are authorized
	DefaultAuthorizedChannels = []string{
		"channel-0", // Osmosis
		"channel-3", // Cosmos Hub
	}
	DefaultEVMChannels = []string{
		"channel-2", // Injective
	}
	DefaultEnableClaims     = true
	DefaultAirdropStartTime = time.Time{}
)

// ParamsKey store key for params
var ParamsKey = []byte("Params")

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params object
func NewParams(
	enableClaim bool,
	claimsDenom string,
	airdropStartTime time.Time,
	durationUntilDecay,
	durationOfDecay time.Duration,
	authorizedChannels,
	evmChannels []string,
) Params {
	return Params{
		EnableClaims:       enableClaim,
		ClaimsDenom:        claimsDenom,
		AirdropStartTime:   airdropStartTime,
		DurationUntilDecay: durationUntilDecay,
		DurationOfDecay:    durationOfDecay,
		AuthorizedChannels: authorizedChannels,
		EVMChannels:        evmChannels,
	}
}

// DefaultParams creates a parameter instance with default values
// for the claims module.
func DefaultParams() Params {
	return Params{
		EnableClaims:       DefaultEnableClaims,
		ClaimsDenom:        DefaultClaimsDenom,
		AirdropStartTime:   DefaultAirdropStartTime,
		DurationUntilDecay: DefaultDurationUntilDecay,
		DurationOfDecay:    DefaultDurationOfDecay,
		AuthorizedChannels: DefaultAuthorizedChannels,
		EVMChannels:        DefaultEVMChannels,
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyEnableClaims, &p.EnableClaims, func(_ interface{}) error { return nil }),
		paramtypes.NewParamSetPair(KeyClaimsDenom, &p.ClaimsDenom, validateDenom),
		paramtypes.NewParamSetPair(KeyAirdropStartTime, &p.AirdropStartTime, func(_ interface{}) error { return nil }),
		paramtypes.NewParamSetPair(KeyDurationUntilDecay, &p.DurationUntilDecay, validateDurationUntilDecay),
		paramtypes.NewParamSetPair(KeyDurationOfDecay, &p.DurationOfDecay, validateDurationOfDecay),
		paramtypes.NewParamSetPair(KeyAuthorizedChannels, &p.AuthorizedChannels, ValidateChannels),
		paramtypes.NewParamSetPair(KeyEVMChannels, &p.AuthorizedChannels, ValidateChannels),
	}
}

// ValidateChannels checks if channels ids are valid
func ValidateChannels(i interface{}) error {
	channels, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, channel := range channels {
		if err := host.ChannelIdentifierValidator(channel); err != nil {
			return errorsmod.Wrap(
				channeltypes.ErrInvalidChannelIdentifier, err.Error(),
			)
		}
	}

	return nil
}

func (p Params) Validate() error {
	if err := validateDurationOfDecay(p.DurationOfDecay); err != nil {
		return err
	}
	if err := validateDurationUntilDecay(p.DurationUntilDecay); err != nil {
		return err
	}
	if err := validateDenom(p.ClaimsDenom); err != nil {
		return err
	}
	if err := ValidateChannels(p.AuthorizedChannels); err != nil {
		return err
	}
	return ValidateChannels(p.EVMChannels)
}

// DecayStartTime returns the time at which the Decay period starts
func (p Params) DecayStartTime() time.Time {
	return p.AirdropStartTime.Add(p.DurationUntilDecay)
}

// AirdropEndTime returns the time at which no further claims will be processed.
func (p Params) AirdropEndTime() time.Time {
	return p.AirdropStartTime.Add(p.DurationUntilDecay).Add(p.DurationOfDecay)
}

// IsClaimsActive returns true if the claiming process is active:
// - claims are enabled AND
// - block time is equal or after the airdrop start time AND
// - block time is before or equal the airdrop end time
func (p Params) IsClaimsActive(blockTime time.Time) bool {
	if !p.EnableClaims || blockTime.Before(p.AirdropStartTime) || blockTime.After(p.AirdropEndTime()) {
		return false
	}
	return true
}

// IsAuthorizedChannel returns true if the channel provided is in the list of
// authorized channels
func (p Params) IsAuthorizedChannel(channel string) bool {
	for _, authorizedChannel := range p.AuthorizedChannels {
		if channel == authorizedChannel {
			return true
		}
	}

	return false
}

// IsEVMChannel returns true if the channel provided is in the list of
// EVM channels
func (p Params) IsEVMChannel(channel string) bool {
	for _, evmChannel := range p.EVMChannels {
		if channel == evmChannel {
			return true
		}
	}

	return false
}

func validateDenom(i interface{}) error {
	denom, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return sdk.ValidateDenom(denom)
}

func validateDurationOfDecay(i interface{}) error {
	duration, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if duration <= 0 {
		return fmt.Errorf("duration of decay must be positive: %d", duration)
	}

	return nil
}

func validateDurationUntilDecay(i interface{}) error {
	duration, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if duration <= 0 {
		return fmt.Errorf("duration util decay must be positive: %d", duration)
	}

	return nil
}
