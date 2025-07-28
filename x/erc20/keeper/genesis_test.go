package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/evmos/evmos/v12/contracts"
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

func (suite *KeeperTestSuite) TestAutoConvertBankToERC20OnGenesis() {
	// Create test accounts with bank balances
	testDenom := "utest"
	testAmount := int64(1000000)

	// Create test metadata
	metadata := banktypes.Metadata{
		Base:        testDenom,
		Display:     "test",
		Name:        "Test Token",
		Symbol:      "TEST",
		Description: "Test token for auto-conversion",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: testDenom, Exponent: 0},
			{Denom: "test", Exponent: 18},
		},
	}

	// Set the metadata in bank keeper
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metadata)

	// Create test accounts and fund them with bank tokens
	account1 := sdk.AccAddress("account1____________")
	account2 := sdk.AccAddress("account2____________")

	// Fund accounts with bank tokens
	coins1 := sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(testAmount)))
	coins2 := sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(testAmount*2)))

	err := suite.app.BankKeeper.MintCoins(suite.ctx, "erc20", coins1)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "erc20", account1, coins1)
	suite.Require().NoError(err)

	err = suite.app.BankKeeper.MintCoins(suite.ctx, "erc20", coins2)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, "erc20", account2, coins2)
	suite.Require().NoError(err)

	// Verify initial bank balances
	balance1Before := suite.app.BankKeeper.GetBalance(suite.ctx, account1, testDenom)
	balance2Before := suite.app.BankKeeper.GetBalance(suite.ctx, account2, testDenom)

	suite.Require().Equal(testAmount, balance1Before.Amount.Int64())
	suite.Require().Equal(testAmount*2, balance2Before.Amount.Int64())

	// Create a token pair with DeployedContractOnGenesisAddr to trigger auto-conversion
	tokenPair := types.TokenPair{
		Erc20Address: types.DeployedContractOnGenesisAddr,
		Denom:        testDenom,
		Enabled:      true,
	}

	// Create genesis state with the token pair
	genesisState := types.GenesisState{
		Params:     types.DefaultParams(),
		TokenPairs: []types.TokenPair{tokenPair},
	}

	// Initialize genesis (this should trigger auto-conversion)
	suite.app.Erc20Keeper.InitGenesis(suite.ctx, genesisState)

	// Verify that bank balances are now zero (converted to ERC20)
	balance1After := suite.app.BankKeeper.GetBalance(suite.ctx, account1, testDenom)
	balance2After := suite.app.BankKeeper.GetBalance(suite.ctx, account2, testDenom)

	suite.Require().True(balance1After.IsZero(), "Account1 bank balance should be zero after conversion")
	suite.Require().True(balance2After.IsZero(), "Account2 bank balance should be zero after conversion")

	// Get the registered token pair to check the deployed contract
	id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, testDenom)
	suite.Require().NotEmpty(id, "Token pair ID should exist")

	registeredPair, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
	suite.Require().True(found, "Token pair should be registered")
	suite.Require().NotEqual(types.DeployedContractOnGenesisAddr, registeredPair.Erc20Address, "Contract should be deployed")

	// Verify ERC20 balances using BalanceOf
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := registeredPair.GetERC20Contract()

	// Check ERC20 balance for account1
	account1EthAddr := common.BytesToAddress(account1)
	balance1ERC20 := suite.app.Erc20Keeper.BalanceOf(suite.ctx, erc20, contract, account1EthAddr)
	suite.Require().NotNil(balance1ERC20, "Account1 ERC20 balance should not be nil")
	suite.Require().Equal(testAmount, balance1ERC20.Int64(), "Account1 should have correct ERC20 balance")

	// Check ERC20 balance for account2
	account2EthAddr := common.BytesToAddress(account2)
	balance2ERC20 := suite.app.Erc20Keeper.BalanceOf(suite.ctx, erc20, contract, account2EthAddr)
	suite.Require().NotNil(balance2ERC20, "Account2 ERC20 balance should not be nil")
	suite.Require().Equal(testAmount*2, balance2ERC20.Int64(), "Account2 should have correct ERC20 balance")
}
