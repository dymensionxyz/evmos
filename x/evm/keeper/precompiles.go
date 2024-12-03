package keeper

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/evmos/evmos/v12/x/evm/statedb"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	corevm "github.com/ethereum/go-ethereum/core/vm"
)

/*
This implementation is limited to readonly, means Precompiled contract is only Allowed to Read from context, any write will cause serious issues to state.

Any write to state will cause serious issues.
Any write to state will cause serious issues.
Any write to state will cause serious issues.
*/

const (
	cpcAddrNonceBech32 byte = iota + 2
)

// CpcBech32FixedAddress is the address of the bech32 custom precompiled contract.
var CpcBech32FixedAddress common.Address

var _ corevm.CustomPrecompiledContractMethodExecutorI = &customPrecompiledContractMethodExecutorImpl{}

func NewCustomPrecompiledContractMethod(
	executor ExtendedCustomPrecompiledContractMethodExecutorI,
) corevm.CustomPrecompiledContractMethod {
	return corevm.CustomPrecompiledContractMethod{
		Method4BytesSignatures: executor.Method4BytesSignatures(),
		RequireGas:             executor.RequireGas(),
		ReadOnly:               executor.ReadOnly(),
		Executor: &customPrecompiledContractMethodExecutorImpl{
			executor: executor,
		},
	}
}

type customPrecompiledContractMethodExecutorImpl struct {
	executor ExtendedCustomPrecompiledContractMethodExecutorI
}

func (m customPrecompiledContractMethodExecutorImpl) Execute(caller corevm.ContractRef, contractAddress common.Address, input []byte, evm *corevm.EVM) ([]byte, error) {
	if input == nil || len(input) < 4 {
		// caller's fault
		panic("invalid call input, minimum 4 bytes required")
	} else if sig := input[:4]; !bytes.Equal(sig, m.executor.Method4BytesSignatures()) {
		// caller's fault
		panic(fmt.Sprintf(
			"mis-match signature, expected %s, got %s",
			hex.EncodeToString(m.executor.Method4BytesSignatures()), hex.EncodeToString(sig),
		))
	}

	ctx := evm.StateDB.(statedb.StateDbWithExt).ExposeSdkContext()
	{ // branch to avoid misuse of ctx
		ctx, _ = ctx.CacheContext()
	}
	return m.executor.Execute(caller, contractAddress, input, CpcExecutorEnv{
		Ctx: ctx,
		Evm: evm,
	})
}

type CustomPrecompiledContractI interface {
	GetName() string
	GetAddress() common.Address
	GetMethodExecutors() []ExtendedCustomPrecompiledContractMethodExecutorI
}

type CpcExecutorEnv struct {
	Ctx sdk.Context
	Evm *corevm.EVM
}

type ExtendedCustomPrecompiledContractMethodExecutorI interface {
	// Execute executes the method with the given input and environment then returns the output.
	Execute(caller corevm.ContractRef, contractAddress common.Address, input []byte, env CpcExecutorEnv) ([]byte, error)

	// Metadata

	Method4BytesSignatures() []byte
	RequireGas() uint64
	ReadOnly() bool
}

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &notSupportedCustomPrecompiledContractMethodExecutor{}

type notSupportedCustomPrecompiledContractMethodExecutor struct {
	method4BytesSignatures []byte
	readOnly               bool
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ CpcExecutorEnv) ([]byte, error) {
	return nil, fmt.Errorf("not supported")
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) Method4BytesSignatures() []byte {
	return n.method4BytesSignatures
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) RequireGas() uint64 {
	if n.readOnly {
		return 0
	}
	return 2
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) ReadOnly() bool {
	return n.readOnly
}

// register
type registeredCustomPrecompiledContract struct {
	// contract is the custom precompiled contract
	contract CustomPrecompiledContractI
	// enableAtVersion is the version at which the custom precompiled contract is enabled
	enableAtVersion uint32
}

var registeredCustomPrecompiledContracts = make(map[common.Address]registeredCustomPrecompiledContract)

// RegisterCustomPrecompiledContract registers a custom precompiled contract with the given address and enable version.
func RegisterCustomPrecompiledContract(contract CustomPrecompiledContractI, enableAtVersion uint32) {
	addr := contract.GetAddress()
	if _, ok := registeredCustomPrecompiledContracts[addr]; ok {
		panic(fmt.Sprintf("custom precompiled contract %s already registered", addr.Hex()))
	}
	registeredCustomPrecompiledContracts[addr] = registeredCustomPrecompiledContract{
		contract:        contract,
		enableAtVersion: enableAtVersion,
	}
}

// GetCustomPrecompiledContractsAtVersion returns the custom precompiled contracts at the given version.
func GetCustomPrecompiledContractsAtVersion(version uint32) []CustomPrecompiledContractI {
	var contracts []CustomPrecompiledContractI
	for _, r := range registeredCustomPrecompiledContracts {
		if r.enableAtVersion <= version {
			contracts = append(contracts, r.contract)
		}
	}

	// apply some extra sorting so everything is deterministic
	slices.SortFunc(contracts, func(i, j CustomPrecompiledContractI) int {
		return bytes.Compare(i.GetAddress().Bytes(), j.GetAddress().Bytes())
	})

	return contracts
}

func init() {
	generatedCpcAddresses := make(map[common.Address]struct{})

	// generateCpcAddress generates a custom precompiled contract address based on the contract address nonce.
	generateCpcAddress := func(contractAddrNonce byte) common.Address {
		if contractAddrNonce == 0 {
			panic("contract address nonce cannot be zero")
		}
		bz := make([]byte, 20)
		bz[0] = 0xCC
		bz[1] = contractAddrNonce
		bz[19] = contractAddrNonce

		addr := common.BytesToAddress(bz)
		if _, ok := generatedCpcAddresses[addr]; ok {
			panic(fmt.Sprintf("generated address %s already exists", addr.Hex()))
		}
		generatedCpcAddresses[addr] = struct{}{}

		return addr
	}

	CpcBech32FixedAddress = generateCpcAddress(cpcAddrNonceBech32)

	RegisterCustomPrecompiledContract(NewBech32CustomPrecompiledContract(), 0)
}
