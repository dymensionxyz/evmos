package keeper_test

import (
	"cosmossdk.io/math"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/evmos/evmos/v12/x/erc20/types"
)

func (suite *KeeperTestSuite) TestERC20InitGenesis() {
	testCases := []struct {
		name         string
		genesisState types.GenesisState
		expectedErr  bool
	}{
		{
			"empty genesis - expected error",
			types.GenesisState{},
			true,
		},
		{
			"default genesis",
			*types.DefaultGenesisState(),
			false,
		},
		{
			"custom genesis",
			types.NewGenesisState(
				types.DefaultParams(),
				[]types.TokenPair{
					{
						Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ",
						Denom:         "coin",
						Enabled:       true,
						ContractOwner: types.OWNER_MODULE,
					},
				}),
			false,
		},
	}

	for _, tc := range testCases {
		if tc.expectedErr {
			suite.Require().Panics(func() {
				suite.app.Erc20Keeper.InitGenesis(suite.ctx, tc.genesisState)
			})
			continue
		}

		suite.Require().NotPanics(func() {
			suite.app.Erc20Keeper.InitGenesis(suite.ctx, tc.genesisState)
		})
		params := suite.app.Erc20Keeper.GetParams(suite.ctx)

		tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
		suite.Require().Equal(tc.genesisState.Params, params)
		if len(tokenPairs) > 0 {
			suite.Require().Equal(tc.genesisState.TokenPairs, tokenPairs)
		} else {
			suite.Require().Len(tc.genesisState.TokenPairs, 0)
		}
	}
}

func (suite *KeeperTestSuite) TestErc20ExportGenesis() {
	customParams := types.DefaultParams()
	customParams.RegistrationFee = math.NewIntWithDecimal(100, 18)

	testGenCases := []struct {
		name         string
		genesisState types.GenesisState
	}{
		{
			"default genesis",
			*types.DefaultGenesisState(),
		},
		{
			"custom genesis",
			types.NewGenesisState(
				customParams,
				[]types.TokenPair{
					{
						Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ",
						Denom:         "coin",
						Enabled:       true,
						ContractOwner: types.OWNER_MODULE,
					},
				}),
		},
	}

	for _, tc := range testGenCases {
		suite.app.Erc20Keeper.InitGenesis(suite.ctx, tc.genesisState)
		suite.Require().NotPanics(func() {
			genesisExported := suite.app.Erc20Keeper.ExportGenesis(suite.ctx)
			params := suite.app.Erc20Keeper.GetParams(suite.ctx)
			suite.Require().Equal(genesisExported.Params, params)

			tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
			if len(tokenPairs) > 0 {
				suite.Require().Equal(genesisExported.TokenPairs, tokenPairs)
				suite.Require().Equal(genesisExported.Params.RegistrationFee, customParams.RegistrationFee)
			} else {
				suite.Require().Len(genesisExported.TokenPairs, 0)
			}
		})
		// }
	}
}

func (s *KeeperTestSuite) TestInitGenesisWithEmptyErc20Address() {
	s.SetupTest()
	metadata := banktypes.Metadata{
		Base:    "adummy",
		Display: "dummy",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "adummy",
				Exponent: 0,
			},
			{
				Denom:    "dummy",
				Exponent: 18,
			},
		},
		Name:   "dummy",
		Symbol: "dummy",
	}

	s.app.BankKeeper.SetDenomMetaData(s.ctx, metadata)

	// Set up the genesis state with an empty ERC20 address
	sentinelAddr := types.DeployedContractOnGenesisAddr
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		TokenPairs: []types.TokenPair{
			{
				Erc20Address:  sentinelAddr,
				Denom:         "adummy",
				Enabled:       true,
				ContractOwner: types.OWNER_UNSPECIFIED,
			},
		},
	}

	err := genesisState.Validate()
	s.Require().NoError(err)

	s.Require().NotPanics(func() {
		s.app.Erc20Keeper.InitGenesis(s.ctx, genesisState)
	})

	id := s.app.Erc20Keeper.GetDenomMap(s.ctx, "adummy")
	pair, found := s.app.Erc20Keeper.GetTokenPair(s.ctx, id)
	s.Require().True(found)
	s.Require().NotEqual(sentinelAddr, pair.Erc20Address) // make sure we actually deploy a new contract
}
