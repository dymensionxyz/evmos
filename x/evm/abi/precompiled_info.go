package abi

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type CustomPrecompiledContractInfo struct {
	Name string
	ABI  abi.ABI
}

func (s *CustomPrecompiledContractInfo) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &s.ABI); err != nil {
		return fmt.Errorf("failed to unmarshal ABI: %w", err)
	}
	return nil
}

func (s CustomPrecompiledContractInfo) UnpackMethodInput(methodName string, fullInput []byte) ([]interface{}, error) {
	return s.findMethodWithSignatureCheck(methodName, fullInput).Inputs.Unpack(fullInput[4:])
}

func (s CustomPrecompiledContractInfo) PackMethodOutput(methodName string, args ...any) ([]byte, error) {
	return s.findMethod(methodName).Outputs.Pack(args...)
}

// findMethodWithSignatureCheck finds a method by name, panic if not exists, panic if the input signature does not match the method signature
func (s CustomPrecompiledContractInfo) findMethodWithSignatureCheck(methodName string, fullInput []byte) abi.Method {
	method := s.findMethod(methodName)
	inputSig := fullInput[:4]
	if !bytes.Equal(method.ID, inputSig) {
		panic(fmt.Sprintf("signature not match for %s: 0x%s != 0x%s", method.Sig, hex.EncodeToString(method.ID), hex.EncodeToString(inputSig)))
	}
	return method
}

// findMethod finds a method by name and panics if it does not exist
func (s CustomPrecompiledContractInfo) findMethod(methodName string) abi.Method {
	method, found := s.ABI.Methods[methodName]
	if !found {
		panic(fmt.Sprintf("method could not be found in %s: %s", s.Name, methodName))
	}
	return method
}

var (
	//go:embed bech32.abi.json
	bech32Json []byte

	Bech32CpcInfo CustomPrecompiledContractInfo
)

func init() {
	var err error

	err = json.Unmarshal(bech32Json, &Bech32CpcInfo)
	if err != nil {
		panic(err)
	}
	Bech32CpcInfo.Name = "Bech32"
}
