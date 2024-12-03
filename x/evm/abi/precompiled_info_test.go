package abi

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

//goland:noinspection GoUnusedGlobalVariable
var (
	bigIntMaxInt64    = new(big.Int).SetUint64(math.MaxInt64)
	bigIntMaxInt64Bz  = common.BytesToHash(bigIntMaxInt64.Bytes()).Bytes()
	bigIntMaxUint64   = new(big.Int).SetUint64(math.MaxUint64)
	bigIntMaxUint64Bz = common.BytesToHash(bigIntMaxUint64.Bytes()).Bytes()
	bigIntOneBz       = common.BytesToHash(big.NewInt(1).Bytes()).Bytes()
	text              = "hello"
	textAbiEncodedBz  = []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
	maxUint8Value     = uint8(math.MaxUint8)
	maxUint8ValueBz   = common.BytesToHash([]byte{math.MaxUint8}).Bytes()
	_32Bytes          = [32]byte{0x1, 0x2, 0x3, 0x32, 0xFF}
)

func TestCustomPrecompiledContractInfo_UnpackMethodInput(t *testing.T) {
	t.Run("pass - can unpack method input", func(t *testing.T) {
		bz, err := Bech32CpcInfo.ABI.Methods["bech32EncodeAddress"].Inputs.Pack(text, common.BytesToAddress([]byte("account")))
		require.NoError(t, err)

		ret, err := Bech32CpcInfo.UnpackMethodInput(
			"bech32EncodeAddress",
			append([]byte{0xb3, 0x61, 0xcf, 0xef}, bz...),
		)

		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, text, ret[0].(string))
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[1].(common.Address))
	})

	t.Run("fail - can not unpack bad method input, less params than expected", func(t *testing.T) {
		_, err := Bech32CpcInfo.UnpackMethodInput(
			"bech32EncodeAddress",
			simpleBuildMethodInput(
				append([]byte{0xb3, 0x61, 0xcf, 0xef}),
			),
		)
		require.Error(t, err)
	})

	t.Run("pass - can unpack bad method input, more params than expected", func(t *testing.T) {
		bz, err := Bech32CpcInfo.ABI.Methods["bech32EncodeAddress"].Inputs.Pack(text, common.BytesToAddress([]byte("account")))
		require.NoError(t, err)

		bz = append(bz, common.BytesToAddress([]byte("another")).Bytes()...)

		ret, err := Bech32CpcInfo.UnpackMethodInput(
			"bech32EncodeAddress",
			append([]byte{0xb3, 0x61, 0xcf, 0xef}, bz...),
		)

		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, text, ret[0].(string))
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[1].(common.Address))
	})

	t.Run("fail - panic if method name could not be found", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = Bech32CpcInfo.UnpackMethodInput(
				"void",
				simpleBuildMethodInput(
					[]byte{0x01, 0x02, 0x03, 0x04},
				),
			)
		})
	})

	t.Run("fail - panic if signature does not match", func(t *testing.T) {
		bz, err := Bech32CpcInfo.ABI.Methods["bech32EncodeAddress"].Inputs.Pack(text, common.BytesToAddress([]byte("account")))
		require.NoError(t, err)

		require.Panics(t, func() {
			_, _ = Bech32CpcInfo.UnpackMethodInput(
				"bech32EncodeAddress",
				append([]byte{0x01, 0x02, 0x03, 0x04}, bz...),
			)
		})
	})
}

func TestCustomPrecompiledContractInfo_PackMethodOutput(t *testing.T) {
	t.Run("pass - can pack method output", func(t *testing.T) {
		ret, err := Bech32CpcInfo.PackMethodOutput(
			"bech32AccountAddrPrefix",
			text,
		)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, ret)
	})

	t.Run("fail - can not pack bad method output, less params than expected", func(t *testing.T) {
		_, err := Bech32CpcInfo.PackMethodOutput(
			"bech32AccountAddrPrefix",
		)
		require.Error(t, err)
	})

	t.Run("fail - can not pack bad method output, more params than expected", func(t *testing.T) {
		_, err := Bech32CpcInfo.PackMethodOutput(
			"bech32AccountAddrPrefix",
			text,
			bigIntMaxUint64,
		)
		require.Error(t, err)
	})

	t.Run("fail - can not pack bad method output, mis-match type", func(t *testing.T) {
		_, err := Bech32CpcInfo.PackMethodOutput(
			"bech32AccountAddrPrefix",
			bigIntMaxUint64, // not string
		)
		require.Error(t, err)
	})

	t.Run("fail - panic if method name could not be found", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = Bech32CpcInfo.PackMethodOutput(
				"void",
			)
		})
		require.Panics(t, func() {
			_, _ = Bech32CpcInfo.PackMethodOutput(
				"void",
				"arg",
			)
		})
	})
}

func Test_Bech32(t *testing.T) {
	cpcInfo := Bech32CpcInfo

	t.Run("bech32EncodeAddress(string,address)", func(t *testing.T) {
		bz, err := cpcInfo.ABI.Methods["bech32EncodeAddress"].Inputs.Pack(text, common.BytesToAddress([]byte("account")))
		require.NoError(t, err)

		ret, err := cpcInfo.UnpackMethodInput(
			"bech32EncodeAddress",
			append([]byte{0xb3, 0x61, 0xcf, 0xef}, bz...),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, text, ret[0].(string))
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[1].(common.Address))

		bz, err = cpcInfo.PackMethodOutput("bech32EncodeAddress", text, true)
		require.NoError(t, err)
		ops, err := cpcInfo.ABI.Methods["bech32EncodeAddress"].Outputs.Unpack(bz)
		require.NoError(t, err)
		require.Len(t, ops, 2)
		require.Equal(t, text, ops[0].(string))
		require.Equal(t, true, ops[1].(bool))
	})
	t.Run("bech32Encode32BytesAddress(string,bytes32)", func(t *testing.T) {
		bz, err := cpcInfo.ABI.Methods["bech32Encode32BytesAddress"].Inputs.Pack(text, _32Bytes)
		require.NoError(t, err)

		ret, err := cpcInfo.UnpackMethodInput(
			"bech32Encode32BytesAddress",
			append([]byte{0xa9, 0x4b, 0x84, 0xb3}, bz...),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, text, ret[0].(string))
		require.Equal(t, _32Bytes, ret[1].([32]byte))

		bz, err = cpcInfo.PackMethodOutput("bech32Encode32BytesAddress", text, true)
		require.NoError(t, err)
		ops, err := cpcInfo.ABI.Methods["bech32Encode32BytesAddress"].Outputs.Unpack(bz)
		require.NoError(t, err)
		require.Len(t, ops, 2)
		require.Equal(t, text, ops[0].(string))
		require.Equal(t, true, ops[1].(bool))
	})
	t.Run("bech32EncodeBytes(string,bytes)", func(t *testing.T) {
		bz, err := cpcInfo.ABI.Methods["bech32EncodeBytes"].Inputs.Pack(text, []byte("buffer"))
		require.NoError(t, err)

		ret, err := cpcInfo.UnpackMethodInput(
			"bech32EncodeBytes",
			append([]byte{0xf6, 0xe0, 0xd5, 0x03}, bz...),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, text, ret[0].(string))
		require.Equal(t, []byte("buffer"), ret[1].([]byte))

		bz, err = cpcInfo.PackMethodOutput("bech32EncodeBytes", text, true)
		require.NoError(t, err)
		ops, err := cpcInfo.ABI.Methods["bech32EncodeBytes"].Outputs.Unpack(bz)
		require.NoError(t, err)
		require.Len(t, ops, 2)
		require.Equal(t, text, ops[0].(string))
		require.Equal(t, true, ops[1].(bool))
	})
	t.Run("bech32Decode(string)", func(t *testing.T) {
		bz, err := cpcInfo.ABI.Methods["bech32Decode"].Inputs.Pack(text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)

		ret, err := cpcInfo.UnpackMethodInput(
			"bech32Decode",
			append([]byte{0xbc, 0x42, 0x53, 0x7f}, bz...),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, text, ret[0].(string))

		bz, err = cpcInfo.PackMethodOutput("bech32Decode", text, bigIntMaxInt64Bz, true)
		require.NoError(t, err)
		ops, err := cpcInfo.ABI.Methods["bech32Decode"].Outputs.Unpack(bz)
		require.NoError(t, err)
		require.Len(t, ops, 3)
		require.Equal(t, text, ops[0].(string))
		require.Equal(t, bigIntMaxInt64Bz, ops[1].([]byte))
		require.Equal(t, true, ops[2].(bool))
	})
	t.Run("bech32AccountAddrPrefix()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("bech32AccountAddrPrefix", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("bech32ValidatorAddrPrefix()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("bech32ValidatorAddrPrefix", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("bech32ConsensusAddrPrefix()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("bech32ConsensusAddrPrefix", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("bech32AccountPubPrefix()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("bech32AccountPubPrefix", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("bech32ValidatorPubPrefix()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("bech32ValidatorPubPrefix", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("bech32ConsensusPubPrefix()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("bech32ConsensusPubPrefix", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
}

func simpleBuildMethodInput(sig []byte, args ...any) []byte {
	if len(sig) != 4 {
		panic("signature must be 4 bytes")
	}

	ret := make([]byte, 0, 4+len(args)*32)
	ret = append(ret, sig...)

	for i, arg := range args {
		if addr, isAddr := arg.(common.Address); isAddr {
			ret = append(ret, make([]byte, 12)...)
			ret = append(ret, addr.Bytes()...)
		} else if vBi, isBigInt := arg.(*big.Int); isBigInt {
			bz := vBi.Bytes()
			ret = append(ret, make([]byte, 32-len(bz))...)
			ret = append(ret, bz...)
		} else {
			panic(fmt.Sprintf("unsupported type %T at %d: %v", arg, i, arg))
		}
	}

	fmt.Println("0x" + hex.EncodeToString(ret))
	return ret
}
