package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/evmos/evmos/v12/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v12/types"
)

func (s *KeeperTestSuite) TestConvertAddress() {
	s.Run("eth to sdk", func() {
		ethAddr := RandomETHAddress(s.T())

		sdkAddr := sdk.AccAddress(ethAddr.Bytes())
		err := sdk.VerifyAddressFormat(sdkAddr.Bytes())
		s.Require().NoError(err)

		ethAddr1 := common.BytesToAddress(sdkAddr.Bytes())

		s.Require().Equal(ethAddr, ethAddr1)
	})

	s.Run("sdk to eth", func() {
		sdkAddr := RandomSDKAccount()

		ethAddr := common.BytesToAddress(sdkAddr.Bytes())
		err := types.ValidateNonZeroAddress(ethAddr.String())
		s.Require().NoError(err)

		sdkAddr1 := sdk.AccAddress(ethAddr.Bytes())

		s.Require().Equal(sdkAddr, sdkAddr1)
	})
}

func RandomSDKAccount() sdk.AccAddress {
	pk := ed25519.GenPrivKey().PubKey()
	return sdk.AccAddress(pk.Address())
}

func RandomETHAddress(t *testing.T) common.Address {
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	return common.BytesToAddress(priv.PubKey().Address().Bytes())
}
