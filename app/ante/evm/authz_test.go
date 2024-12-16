package evm_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/common"

	evmante "github.com/evmos/evmos/v12/app/ante/evm"
	"github.com/evmos/evmos/v12/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v12/testutil"
	evmtypes "github.com/evmos/evmos/v12/x/evm/types"
)

func (s *AnteTestSuite) TestAuthorizationDecorator() {
	priv, err := ethsecp256k1.GenerateKey()
	s.Require().NoError(err)
	addr := common.BytesToAddress(priv.PubKey().Address().Bytes())

	testCases := []struct {
		name      string
		onBehalf  *common.Address
		grantAuth bool
		expPass   bool
	}{
		{
			name:      "success: empty OnBehalf",
			onBehalf:  nil,
			grantAuth: false,
			expPass:   true,
		},
		{
			name:      "success: non-empty OnBehalf with access granted",
			onBehalf:  &addr,
			grantAuth: true,
			expPass:   true,
		},
		{
			name:      "error: non-empty OnBehalf with access not granted",
			onBehalf:  &addr,
			grantAuth: false,
			expPass:   false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			// Setup MsgEthereumTx
			txParams := &evmtypes.EvmTxArgs{
				ChainID:  s.app.EvmKeeper.ChainID(),
				Nonce:    1,
				Amount:   big.NewInt(10),
				GasLimit: 1000,
				GasPrice: big.NewInt(1),
			}
			tx := evmtypes.NewTx(txParams)
			tx.From = s.address.Hex()
			if tc.onBehalf != nil {
				tx.OnBehalf = tc.onBehalf.Hex()
			}

			// Grant authorization if required
			if tc.grantAuth {
				grantee := sdk.AccAddress(tc.onBehalf.Bytes())
				granter := sdk.AccAddress(s.address.Bytes())
				a := authz.NewGenericAuthorization(sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}))
				err = s.app.AuthzKeeper.SaveGrant(s.ctx, grantee, granter, a, nil)
				s.Require().NoError(err)
			}

			// Run AnteHandle
			d := evmante.NewAuthorizationDecorator(s.app.AuthzKeeper)
			ctx, err := d.AnteHandle(s.ctx, tx, false, testutil.NextFn)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(ctx)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
