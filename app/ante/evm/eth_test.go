package evm_test

import (
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	ethante "github.com/evmos/evmos/v12/app/ante/evm"
	"github.com/evmos/evmos/v12/contracts"
	"github.com/evmos/evmos/v12/server/config"
	"github.com/evmos/evmos/v12/testutil"
	testutiltx "github.com/evmos/evmos/v12/testutil/tx"
	"github.com/evmos/evmos/v12/types"
	"github.com/evmos/evmos/v12/utils"
	erc20types "github.com/evmos/evmos/v12/x/erc20/types"
	"github.com/evmos/evmos/v12/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v12/x/evm/types"
	inflationtypes "github.com/evmos/evmos/v12/x/inflation/types"
)

func (suite *AnteTestSuite) TestNewEthAccountVerificationDecorator() {
	dec := ethante.NewEthAccountVerificationDecorator(
		suite.app.AccountKeeper, suite.app.EvmKeeper,
	)

	addr := testutiltx.GenerateAddress()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx.From = addr.Hex()

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		checkTx  bool
		expPass  bool
	}{
		{"not CheckTx", nil, func() {}, false, true},
		{"invalid transaction type", &testutiltx.InvalidTx{}, func() {}, true, false},
		{
			"sender not set to msg",
			tx,
			func() {},
			true,
			false,
		},
		{
			"sender not EOA",
			tx,
			func() {
				// set not as an EOA
				vmdb.SetCode(addr, []byte("1"))
			},
			true,
			false,
		},
		{
			"not enough balance to cover tx cost",
			tx,
			func() {
				// reset back to EOA
				vmdb.SetCode(addr, nil)
			},
			true,
			false,
		},
		{
			"success new account",
			tx,
			func() {
				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			true,
			true,
		},
		{
			"success existing account",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			true,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb = testutil.NewStateDB(suite.ctx, suite.app.EvmKeeper)
			tc.malleate()
			suite.Require().NoError(vmdb.Commit())

			_, err := dec.AnteHandle(suite.ctx.WithIsCheckTx(tc.checkTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthNonceVerificationDecorator() {
	suite.SetupTest()
	dec := ethante.NewEthIncrementSenderSequenceDecorator(suite.app.AccountKeeper)

	addr := testutiltx.GenerateAddress()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx.From = addr.Hex()

	testCases := []struct {
		name      string
		tx        sdk.Tx
		malleate  func()
		reCheckTx bool
		expPass   bool
	}{
		{"ReCheckTx", &testutiltx.InvalidTx{}, func() {}, true, false},
		{"invalid transaction type", &testutiltx.InvalidTx{}, func() {}, false, false},
		{"sender account not found", tx, func() {}, false, false},
		{
			"sender nonce missmatch",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			false,
			false,
		},
		{
			"success",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.Require().NoError(acc.SetSequence(1))
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()
			_, err := dec.AnteHandle(suite.ctx.WithIsReCheckTx(tc.reCheckTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

const ibcBase = "ibc/7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF"

func (suite *AnteTestSuite) TestEthGasConsumeDecorator() {
	chainID := suite.app.EvmKeeper.ChainID()
	dec := ethante.NewEthGasConsumeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.Erc20Keeper, suite.app.DistrKeeper, suite.app.EvmKeeper, suite.app.StakingKeeper, config.DefaultMaxTxGasWanted)

	addr := testutiltx.GenerateAddress()

	txGasLimit := uint64(1000)

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: txGasLimit,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx.From = addr.Hex()

	ethCfg := suite.app.EvmKeeper.GetParams(suite.ctx).
		ChainConfig.EthereumConfig(chainID)
	baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx, ethCfg)
	suite.Require().Equal(int64(1000000000), baseFee.Int64())

	gasPrice := new(big.Int).Add(baseFee, evmtypes.DefaultPriorityReduction.BigInt())

	tx2GasLimit := uint64(1000000)
	eth2TxContractParams := &evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: tx2GasLimit,
		GasPrice: gasPrice,
		Accesses: &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	tx2 := evmtypes.NewTx(eth2TxContractParams)
	tx2.From = addr.Hex()
	tx2Priority := int64(1)

	tx3GasLimit := types.BlockGasLimit(suite.ctx) + uint64(1)
	eth3TxContractParams := &evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: tx3GasLimit,
		GasPrice: gasPrice,
		Accesses: &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	tx3 := evmtypes.NewTx(eth3TxContractParams)

	dynamicTxContractParams := &evmtypes.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  tx2GasLimit,
		GasFeeCap: new(big.Int).Add(baseFee, big.NewInt(evmtypes.DefaultPriorityReduction.Int64()*2)),
		GasTipCap: evmtypes.DefaultPriorityReduction.BigInt(),
		Accesses:  &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	dynamicFeeTx := evmtypes.NewTx(dynamicTxContractParams)
	dynamicFeeTx.From = addr.Hex()
	dynamicFeeTxPriority := int64(1)

	var vmdb *statedb.StateDB

	testCases := []struct {
		name        string
		tx          sdk.Tx
		gasLimit    uint64
		malleate    func(ctx sdk.Context) sdk.Context
		expPass     bool
		expPanic    bool
		expPriority int64
		postCheck   func(ctx sdk.Context)
	}{
		{
			"invalid transaction type",
			&testutiltx.InvalidTx{},
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false,
			false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"sender not found",
			evmtypes.NewTx(&evmtypes.EvmTxArgs{
				ChainID:  chainID,
				Nonce:    1,
				Amount:   big.NewInt(10),
				GasLimit: 1000,
				GasPrice: big.NewInt(1),
			}),
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"gas limit too low",
			tx,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"gas limit above block gas limit",
			tx3,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"not enough balance for fees",
			tx2,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"not enough tx gas",
			tx2,
			0,
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e6))
				return ctx
			},
			false, true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"not enough block gas",
			tx2,
			0,
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e6))
				return ctx.WithBlockGasMeter(sdk.NewGasMeter(1))
			},
			false, true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"success - legacy tx",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e16))
				return ctx.WithBlockGasMeter(sdk.NewGasMeter(1e19))
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {},
		},
		{
			"success - dynamic fee tx",
			dynamicFeeTx,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e16))
				return ctx.WithBlockGasMeter(sdk.NewGasMeter(1e19))
			},
			true, false,
			dynamicFeeTxPriority,
			func(ctx sdk.Context) {},
		},
		{
			"success - gas limit on gasMeter is set on ReCheckTx mode",
			dynamicFeeTx,
			0, // for reCheckTX mode, gas limit should be set to 0
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e15))
				return ctx.WithIsReCheckTx(true)
			},
			true, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"success - legacy tx - insufficient funds but enough staking rewards",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				ctx, err := testutil.PrepareAccountsForDelegationRewards(
					suite.T(), ctx, suite.app, sdk.AccAddress(addr.Bytes()), sdk.ZeroInt(), sdk.NewInt(1e16),
				)
				suite.Require().NoError(err, "error while preparing accounts for delegation rewards")
				return ctx.
					WithBlockGasMeter(sdk.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(addr.Bytes()), utils.BaseDenom)
				suite.Require().False(
					balance.Amount.IsZero(),
					"the fees are paid after withdrawing (a surplus amount of) staking rewards, so it should be higher than the initial balance",
				)

				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.app.DistrKeeper, sdk.AccAddress(addr.Bytes()))
				suite.Require().NoError(err, "error while querying delegation total rewards")
				suite.Require().Nil(rewards, "the total rewards should be nil after withdrawing all of them")
			},
		},
		{
			"success - legacy tx - enough funds so no staking rewards should be used",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				ctx, err := testutil.PrepareAccountsForDelegationRewards(
					suite.T(), ctx, suite.app, sdk.AccAddress(addr.Bytes()), sdk.NewInt(1e16), sdk.NewInt(1e16),
				)
				suite.Require().NoError(err, "error while preparing accounts for delegation rewards")
				return ctx.
					WithBlockGasMeter(sdk.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(addr.Bytes()), utils.BaseDenom)
				suite.Require().True(
					balance.Amount.LT(sdk.NewInt(1e16)),
					"the fees are paid using the available balance, so it should be lower than the initial balance",
				)

				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.app.DistrKeeper, sdk.AccAddress(addr.Bytes()))
				suite.Require().NoError(err, "error while querying delegation total rewards")

				// NOTE: the total rewards should be the same as after the setup, since
				// the fees are paid using the account balance
				suite.Require().Equal(
					sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, sdk.NewInt(1e16))),
					rewards,
					"the total rewards should be the same as after the setup, since the fees are paid using the account balance",
				)
			},
		},
		{
			"success - legacy tx - IBC gas denom",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				// update evm params to use IBC denom as gas denom
				params := suite.app.EvmKeeper.GetParams(ctx)
				params.GasDenom = ibcBase
				err := suite.app.EvmKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				// register IBC denom
				metadataIbc := banktypes.Metadata{
					Description: "ATOM IBC voucher (channel 14)",
					Base:        ibcBase,
					// NOTE: Denom units MUST be increasing
					DenomUnits: []*banktypes.DenomUnit{
						{
							Denom:    ibcBase,
							Exponent: 0,
						},
					},
					Name:    "ATOM channel-14",
					Symbol:  "ibcATOM-14",
					Display: ibcBase,
				}

				// initial IBC denom
				err = suite.app.BankKeeper.MintCoins(ctx, inflationtypes.ModuleName, sdk.Coins{sdk.NewInt64Coin(metadataIbc.Base, 1)})
				suite.Require().NoError(err)

				// register ERC20 representation of IBC denom
				_, err = suite.app.Erc20Keeper.RegisterCoin(ctx, metadataIbc)
				suite.Require().NoError(err)

				// fund sender's SDK account: mint IBC coins and convert it to ERC20 tokens
				coin := sdk.NewCoin(ibcBase, sdk.NewInt(1e16))
				coins := sdk.NewCoins(coin)
				sender := sdk.AccAddress(addr.Bytes())
				err = testutil.FundAccount(ctx, suite.app.BankKeeper, sender, coins)
				suite.Require().NoError(err)

				// check that SDK balance is funded with IBC coins
				senderBalances := suite.app.BankKeeper.GetBalance(ctx, sender, ibcBase)
				suite.Require().True(senderBalances.IsPositive())

				// ensure that the fee collector balance is empty initially
				feeCollectorInitialBalance := suite.app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), ibcBase)
				suite.Require().True(feeCollectorInitialBalance.IsZero())

				return ctx.
					WithBlockGasMeter(sdk.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				// check the fee collector balance, it should be positive (initially it was empty)
				balance := suite.app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), ibcBase)
				suite.Require().True(balance.IsPositive())
			},
		},
		{
			"success - legacy tx - IBC gas denom, not enough SDK coins, enough ERC20 tokens",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				// update evm params to use IBC denom as gas denom
				params := suite.app.EvmKeeper.GetParams(ctx)
				params.GasDenom = ibcBase
				err := suite.app.EvmKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				// register IBC denom
				metadataIbc := banktypes.Metadata{
					Description: "ATOM IBC voucher (channel 14)",
					Base:        ibcBase,
					// NOTE: Denom units MUST be increasing
					DenomUnits: []*banktypes.DenomUnit{
						{
							Denom:    ibcBase,
							Exponent: 0,
						},
					},
					Name:    "ATOM channel-14",
					Symbol:  "ibcATOM-14",
					Display: ibcBase,
				}

				// initial IBC denom
				err = suite.app.BankKeeper.MintCoins(ctx, inflationtypes.ModuleName, sdk.Coins{sdk.NewInt64Coin(metadataIbc.Base, 1)})
				suite.Require().NoError(err)

				// register ERC20 representation of IBC denom
				tp, err := suite.app.Erc20Keeper.RegisterCoin(ctx, metadataIbc)
				suite.Require().NoError(err)

				// fund sender's eth account: mint IBC coins and convert it to ERC20 tokens
				coin := sdk.NewCoin(ibcBase, sdk.NewInt(1e16))
				coins := sdk.NewCoins(coin)
				sender := sdk.AccAddress(addr.Bytes())
				err = testutil.FundAccount(ctx, suite.app.BankKeeper, sender, coins)
				suite.Require().NoError(err)
				msg := erc20types.NewMsgConvertCoin(
					coin,
					common.BytesToAddress(sender.Bytes()),
					sender,
				)
				_, err = suite.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(ctx), msg)
				suite.Require().NoError(err)

				// now the sender's eth balance is funded with ERC20 tokens
				senderBalance := suite.app.Erc20Keeper.BalanceOf(ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, tp.GetERC20Contract(), common.BytesToAddress(addr.Bytes()))
				suite.Require().NotNil(senderBalance)
				suite.Require().Equal(senderBalance.Cmp(big.NewInt(1e16)), 0) // == 1e16

				// while SDK balance is empty
				senderBalances := suite.app.BankKeeper.GetAllBalances(ctx, sender)
				suite.Require().True(senderBalances.IsZero())

				// ensure that the fee collector balance is empty initially
				feeCollectorInitialBalance := suite.app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), ibcBase)
				suite.Require().True(feeCollectorInitialBalance.IsZero())

				return ctx.
					WithBlockGasMeter(sdk.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				// get the token pair for the IBC denom
				tpID := suite.app.Erc20Keeper.GetTokenPairID(ctx, ibcBase)
				suite.Require().NotNil(tpID)
				tp, found := suite.app.Erc20Keeper.GetTokenPair(ctx, tpID)
				suite.Require().True(found)

				// check the sender's ERC20 balance, it should be less than the initial balance
				senderBalance := suite.app.Erc20Keeper.BalanceOf(ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, tp.GetERC20Contract(), common.BytesToAddress(addr.Bytes()))
				suite.Require().NotNil(senderBalance)
				suite.Require().Equal(senderBalance.Cmp(big.NewInt(1e16)), -1) // < 1e16

				// check the fee collector balance, it should be positive (initially it was empty)
				balance := suite.app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), ibcBase)
				suite.Require().True(balance.IsPositive())
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			cacheCtx, _ := suite.ctx.CacheContext()
			// Create new stateDB for each test case from the cached context
			vmdb = testutil.NewStateDB(cacheCtx, suite.app.EvmKeeper)
			cacheCtx = tc.malleate(cacheCtx)
			suite.Require().NoError(vmdb.Commit())

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(cacheCtx.WithIsCheckTx(true).WithGasMeter(sdk.NewGasMeter(1)), tc.tx, false, testutil.NextFn)
				})
				return
			}

			ctx, err := dec.AnteHandle(cacheCtx.WithIsCheckTx(true).WithGasMeter(sdk.NewInfiniteGasMeter()), tc.tx, false, testutil.NextFn)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expPriority, ctx.Priority())
			} else {
				suite.Require().Error(err)
			}
			suite.Require().Equal(tc.gasLimit, ctx.GasMeter().Limit())

			// check state after the test case
			tc.postCheck(ctx)
		})
	}
}

func (suite *AnteTestSuite) TestCanTransferDecorator() {
	dec := ethante.NewCanTransferDecorator(suite.app.EvmKeeper)

	addr, privKey := testutiltx.NewAddrKey()

	suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, big.NewInt(100))
	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:   suite.app.EvmKeeper.ChainID(),
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  1000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(150),
		GasTipCap: big.NewInt(200),
		Accesses:  &ethtypes.AccessList{},
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx2 := evmtypes.NewTx(ethContractCreationTxParams)

	tx.From = addr.Hex()

	err := tx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
	}{
		{"invalid transaction type", &testutiltx.InvalidTx{}, func() {}, false},
		{"AsMessage failed", tx2, func() {}, false},
		{
			"evm CanTransfer failed",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			false,
		},
		{
			"success",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb = testutil.NewStateDB(suite.ctx, suite.app.EvmKeeper)
			tc.malleate()
			suite.Require().NoError(vmdb.Commit())

			_, err := dec.AnteHandle(suite.ctx.WithIsCheckTx(true), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthIncrementSenderSequenceDecorator() {
	dec := ethante.NewEthIncrementSenderSequenceDecorator(suite.app.AccountKeeper)
	addr, privKey := testutiltx.NewAddrKey()

	ethTxContractParamsNonce0 := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    0,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	contract := evmtypes.NewTx(ethTxContractParamsNonce0)
	contract.From = addr.Hex()
	err := contract.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	to := testutiltx.GenerateAddress()
	ethTxParamsNonce0 := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    0,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx := evmtypes.NewTx(ethTxParamsNonce0)
	tx.From = addr.Hex()
	err = tx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	ethTxParamsNonce1 := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx2 := evmtypes.NewTx(ethTxParamsNonce1)
	tx2.From = addr.Hex()
	err = tx2.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
		expPanic bool
	}{
		{
			"invalid transaction type",
			&testutiltx.InvalidTx{},
			func() {},
			false, false,
		},
		{
			"no signers",
			evmtypes.NewTx(ethTxParamsNonce1),
			func() {},
			false, false,
		},
		{
			"account not set to store",
			tx,
			func() {},
			false, false,
		},
		{
			"success - create contract",
			contract,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			true, false,
		},
		{
			"success - call",
			tx2,
			func() {},
			true, false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
				msg := tc.tx.(*evmtypes.MsgEthereumTx)

				txData, err := evmtypes.UnpackTxData(msg.Data)
				suite.Require().NoError(err)

				nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, addr)
				suite.Require().Equal(txData.GetNonce()+1, nonce)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
