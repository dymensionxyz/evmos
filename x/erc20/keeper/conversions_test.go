package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/evmos/evmos/v12/x/erc20/types"
)

func (s *KeeperTestSuite) TestTryConvertErc20Sdk() {
	testCases := []struct {
		name          string
		erc20Balance  int64
		convertAmount int64
		error         error
	}{
		{
			name:          "ok - sufficient funds",
			erc20Balance:  100,
			convertAmount: 50,
			error:         nil,
		},
		{
			name:          "ok - equal funds",
			erc20Balance:  100,
			convertAmount: 100,
			error:         nil,
		},
		{
			name:          "fail - insufficient erc20 balance",
			erc20Balance:  100,
			convertAmount: 200,
			error:         errortypes.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()
			s.setupRegisterCoin(metadataIbc)
			s.Commit()

			// precondition: fund sender's eth account
			coin := sdk.NewCoin(ibcBase, math.NewInt(tc.erc20Balance))
			coins := sdk.NewCoins(coin)
			sender := sdk.AccAddress(s.address.Bytes())
			err := s.app.BankKeeper.MintCoins(s.ctx, types.ModuleName, coins)
			s.Require().NoError(err)
			err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, sender, coins)
			s.Require().NoError(err)
			msg := types.NewMsgConvertCoin(
				coin,
				common.BytesToAddress(sender.Bytes()),
				sender,
			)

			_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.ctx), msg)
			s.Require().NoError(err)
			s.Commit()

			// now sender's eth balance is funded while SDK balance is empty
			// now convert the coin and send it from sender's eth to sdk account
			err = s.app.Erc20Keeper.TryConvertErc20Sdk(s.ctx, sender, sender, ibcBase, math.NewInt(tc.convertAmount))

			if tc.error == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().ErrorIs(err, tc.error)
			}
		})
	}
}
