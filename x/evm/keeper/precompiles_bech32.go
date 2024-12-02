package keeper

import (
	"github.com/cosmos/cosmos-sdk/types/bech32"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/evmos/v12/x/evm/abi"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
)

// contract

var _ CustomPrecompiledContractI = &bech32CustomPrecompiledContract{}

// bech32CustomPrecompiledContract
type bech32CustomPrecompiledContract struct {
	executors []ExtendedCustomPrecompiledContractMethodExecutorI
}

const (
	maxBech32EncodeBufferSizeAllowed = 256
)

// NewBech32CustomPrecompiledContract creates a new bech32 custom precompiled contract.
func NewBech32CustomPrecompiledContract() CustomPrecompiledContractI {
	contract := &bech32CustomPrecompiledContract{}

	contract.executors = []ExtendedCustomPrecompiledContractMethodExecutorI{
		&bech32CustomPrecompiledContractRoEncodeAddress{},
		&bech32CustomPrecompiledContractRoEncode32BytesAddress{},
		&bech32CustomPrecompiledContractRoEncodeBytes{},
		&bech32CustomPrecompiledContractRoDecode{},
		&bech32CustomPrecompiledContractRoAccountAddrPrefix{},
		&bech32CustomPrecompiledContractRoValidatorAddrPrefix{},
		&bech32CustomPrecompiledContractRoConsensusAddrPrefix{},
		&bech32CustomPrecompiledContractRoAccountPubPrefix{},
		&bech32CustomPrecompiledContractRoValidatorPubPrefix{},
		&bech32CustomPrecompiledContractRoConsensusPubPrefix{},
	}

	return contract
}

func (m bech32CustomPrecompiledContract) GetName() string {
	return abi.Bech32CpcInfo.Name
}

func (m bech32CustomPrecompiledContract) GetAddress() common.Address {
	return CpcBech32FixedAddress
}

func (m bech32CustomPrecompiledContract) GetMethodExecutors() []ExtendedCustomPrecompiledContractMethodExecutorI {
	return m.executors
}

// bech32EncodeAddress(string,address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoEncodeAddress{}

type bech32CustomPrecompiledContractRoEncodeAddress struct{}

func (e bech32CustomPrecompiledContractRoEncodeAddress) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.Bech32CpcInfo.UnpackMethodInput("bech32EncodeAddress", input)
	if err != nil {
		return nil, err
	}

	hrp := ips[0].(string)
	address := ips[1].(common.Address)

	result, err := bech32.ConvertAndEncode(hrp, address.Bytes())
	if err != nil {
		return abi.Bech32CpcInfo.PackMethodOutput("bech32EncodeAddress", "", false)
	}

	return abi.Bech32CpcInfo.PackMethodOutput("bech32EncodeAddress", result, true)
}

func (e bech32CustomPrecompiledContractRoEncodeAddress) Method4BytesSignatures() []byte {
	return []byte{0xb3, 0x61, 0xcf, 0xef}
}

func (e bech32CustomPrecompiledContractRoEncodeAddress) RequireGas() uint64 {
	return 30_000
}

func (e bech32CustomPrecompiledContractRoEncodeAddress) ReadOnly() bool {
	return true
}

// bech32Encode32BytesAddress(string,bytes32)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoEncode32BytesAddress{}

type bech32CustomPrecompiledContractRoEncode32BytesAddress struct{}

func (e bech32CustomPrecompiledContractRoEncode32BytesAddress) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.Bech32CpcInfo.UnpackMethodInput("bech32Encode32BytesAddress", input)
	if err != nil {
		return nil, err
	}

	hrp := ips[0].(string)
	address := ips[1].([32]byte)

	result, err := bech32.ConvertAndEncode(hrp, address[:])
	if err != nil {
		return abi.Bech32CpcInfo.PackMethodOutput("bech32Encode32BytesAddress", "", false)
	}

	return abi.Bech32CpcInfo.PackMethodOutput("bech32Encode32BytesAddress", result, true)
}

func (e bech32CustomPrecompiledContractRoEncode32BytesAddress) Method4BytesSignatures() []byte {
	return []byte{0xa9, 0x4b, 0x84, 0xb3}
}

func (e bech32CustomPrecompiledContractRoEncode32BytesAddress) RequireGas() uint64 {
	return 60_000
}

func (e bech32CustomPrecompiledContractRoEncode32BytesAddress) ReadOnly() bool {
	return true
}

// bech32EncodeBytes(string,bytes)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoEncodeBytes{}

type bech32CustomPrecompiledContractRoEncodeBytes struct{}

func (e bech32CustomPrecompiledContractRoEncodeBytes) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.Bech32CpcInfo.UnpackMethodInput("bech32EncodeBytes", input)
	if err != nil {
		return nil, err
	}

	hrp := ips[0].(string)
	buffer := ips[1].([]byte)

	if len(buffer) > maxBech32EncodeBufferSizeAllowed {
		return abi.Bech32CpcInfo.PackMethodOutput("bech32EncodeBytes", "", false)
	}

	result, err := bech32.ConvertAndEncode(hrp, buffer)
	if err != nil {
		return abi.Bech32CpcInfo.PackMethodOutput("bech32EncodeBytes", "", false)
	}

	return abi.Bech32CpcInfo.PackMethodOutput("bech32EncodeBytes", result, true)
}

func (e bech32CustomPrecompiledContractRoEncodeBytes) Method4BytesSignatures() []byte {
	return []byte{0xf6, 0xe0, 0xd5, 0x03}
}

func (e bech32CustomPrecompiledContractRoEncodeBytes) RequireGas() uint64 {
	return 200_000
}

func (e bech32CustomPrecompiledContractRoEncodeBytes) ReadOnly() bool {
	return true
}

// bech32Decode(string)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoDecode{}

type bech32CustomPrecompiledContractRoDecode struct{}

func (e bech32CustomPrecompiledContractRoDecode) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.Bech32CpcInfo.UnpackMethodInput("bech32Decode", input)
	if err != nil {
		return nil, err
	}

	inputBech32 := ips[0].(string)

	hrp, buffer, err := bech32.DecodeAndConvert(inputBech32)
	if err != nil {
		return abi.Bech32CpcInfo.PackMethodOutput("bech32Decode", "", []byte{}, false)
	}

	return abi.Bech32CpcInfo.PackMethodOutput("bech32Decode", hrp, buffer, true)
}

func (e bech32CustomPrecompiledContractRoDecode) Method4BytesSignatures() []byte {
	return []byte{0xbc, 0x42, 0x53, 0x7f}
}

func (e bech32CustomPrecompiledContractRoDecode) RequireGas() uint64 {
	return 200_000
}

func (e bech32CustomPrecompiledContractRoDecode) ReadOnly() bool {
	return true
}

// bech32AccountAddrPrefix()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoAccountAddrPrefix{}

type bech32CustomPrecompiledContractRoAccountAddrPrefix struct{}

func (e bech32CustomPrecompiledContractRoAccountAddrPrefix) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return abi.Bech32CpcInfo.PackMethodOutput("bech32AccountAddrPrefix", sdk.GetConfig().GetBech32AccountAddrPrefix())
}

func (e bech32CustomPrecompiledContractRoAccountAddrPrefix) Method4BytesSignatures() []byte {
	return []byte{0x96, 0x44, 0x3b, 0x16}
}

func (e bech32CustomPrecompiledContractRoAccountAddrPrefix) RequireGas() uint64 {
	return 5000
}

func (e bech32CustomPrecompiledContractRoAccountAddrPrefix) ReadOnly() bool {
	return true
}

// bech32ValidatorAddrPrefix()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoValidatorAddrPrefix{}

type bech32CustomPrecompiledContractRoValidatorAddrPrefix struct{}

func (e bech32CustomPrecompiledContractRoValidatorAddrPrefix) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return abi.Bech32CpcInfo.PackMethodOutput("bech32ValidatorAddrPrefix", sdk.GetConfig().GetBech32ValidatorAddrPrefix())
}

func (e bech32CustomPrecompiledContractRoValidatorAddrPrefix) Method4BytesSignatures() []byte {
	return []byte{0x80, 0x36, 0xb2, 0x25}
}

func (e bech32CustomPrecompiledContractRoValidatorAddrPrefix) RequireGas() uint64 {
	return 5000
}

func (e bech32CustomPrecompiledContractRoValidatorAddrPrefix) ReadOnly() bool {
	return true
}

// bech32ConsensusAddrPrefix()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoConsensusAddrPrefix{}

type bech32CustomPrecompiledContractRoConsensusAddrPrefix struct{}

func (e bech32CustomPrecompiledContractRoConsensusAddrPrefix) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return abi.Bech32CpcInfo.PackMethodOutput("bech32ConsensusAddrPrefix", sdk.GetConfig().GetBech32ConsensusAddrPrefix())
}

func (e bech32CustomPrecompiledContractRoConsensusAddrPrefix) Method4BytesSignatures() []byte {
	return []byte{0x88, 0x33, 0x3d, 0xe6}
}

func (e bech32CustomPrecompiledContractRoConsensusAddrPrefix) RequireGas() uint64 {
	return 5000
}

func (e bech32CustomPrecompiledContractRoConsensusAddrPrefix) ReadOnly() bool {
	return true
}

// bech32AccountPubPrefix()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoAccountPubPrefix{}

type bech32CustomPrecompiledContractRoAccountPubPrefix struct{}

func (e bech32CustomPrecompiledContractRoAccountPubPrefix) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return abi.Bech32CpcInfo.PackMethodOutput("bech32AccountPubPrefix", sdk.GetConfig().GetBech32AccountPubPrefix())
}

func (e bech32CustomPrecompiledContractRoAccountPubPrefix) Method4BytesSignatures() []byte {
	return []byte{0x76, 0x5c, 0x9d, 0x92}
}

func (e bech32CustomPrecompiledContractRoAccountPubPrefix) RequireGas() uint64 {
	return 5000
}

func (e bech32CustomPrecompiledContractRoAccountPubPrefix) ReadOnly() bool {
	return true
}

// bech32ValidatorPubPrefix()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoValidatorPubPrefix{}

type bech32CustomPrecompiledContractRoValidatorPubPrefix struct{}

func (e bech32CustomPrecompiledContractRoValidatorPubPrefix) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return abi.Bech32CpcInfo.PackMethodOutput("bech32ValidatorPubPrefix", sdk.GetConfig().GetBech32ValidatorPubPrefix())
}

func (e bech32CustomPrecompiledContractRoValidatorPubPrefix) Method4BytesSignatures() []byte {
	return []byte{0x73, 0x74, 0xcb, 0x91}
}

func (e bech32CustomPrecompiledContractRoValidatorPubPrefix) RequireGas() uint64 {
	return 5000
}

func (e bech32CustomPrecompiledContractRoValidatorPubPrefix) ReadOnly() bool {
	return true
}

// bech32ConsensusPubPrefix()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &bech32CustomPrecompiledContractRoConsensusPubPrefix{}

type bech32CustomPrecompiledContractRoConsensusPubPrefix struct{}

func (e bech32CustomPrecompiledContractRoConsensusPubPrefix) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return abi.Bech32CpcInfo.PackMethodOutput("bech32ConsensusPubPrefix", sdk.GetConfig().GetBech32ConsensusPubPrefix())
}

func (e bech32CustomPrecompiledContractRoConsensusPubPrefix) Method4BytesSignatures() []byte {
	return []byte{0x2a, 0x99, 0xc3, 0x42}
}

func (e bech32CustomPrecompiledContractRoConsensusPubPrefix) RequireGas() uint64 {
	return 5000
}

func (e bech32CustomPrecompiledContractRoConsensusPubPrefix) ReadOnly() bool {
	return true
}
