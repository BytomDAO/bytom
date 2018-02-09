package instance

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/bytom/encoding/json"
	"github.com/bytom/exp/ivy/compiler"
	"github.com/bytom/protocol/vm"
)

// RevealPreimageBodyBytes refer to contract's body
var RevealPreimageBodyBytes []byte

func init() {
	RevealPreimageBodyBytes, _ = hex.DecodeString("7caa87")
}

// contract RevealPreimage(hash: Hash) locks value
//
// SWAP   [... hash string]
// SHA3   [... hash sha3(string)]
// SWAP   [... sha3(string) hash]
// EQUAL  [... (sha3(string) == hash)]

// PayToRevealPreimage instantiates contract RevealPreimage as a program with specific arguments.
func PayToRevealPreimage(hash []byte) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "hash", Type: "Hash"},
	}
	var _contractArgs []compiler.ContractArg
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&hash)})
	return compiler.Instantiate(RevealPreimageBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToRevealPreimage parses the arguments out of an instantiation of contract RevealPreimage.
// If the input is not an instantiation of RevealPreimage, returns an error.
func ParsePayToRevealPreimage(prog []byte) ([][]byte, error) {
	var result [][]byte
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 1; i++ {
		if len(insts) == 0 {
			return nil, fmt.Errorf("program too short")
		}
		if !insts[0].IsPushdata() {
			return nil, fmt.Errorf("too few arguments")
		}
		result = append(result, insts[0].Data)
		insts = insts[1:]
	}
	if len(insts) != 4 {
		return nil, fmt.Errorf("program too short")
	}
	if insts[0].Op != vm.OP_DEPTH {
		return nil, fmt.Errorf("wrong program format")
	}
	if !insts[1].IsPushdata() {
		return nil, fmt.Errorf("wrong program format")
	}
	if !bytes.Equal(RevealPreimageBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match RevealPreimage")
	}
	if !insts[2].IsPushdata() {
		return nil, fmt.Errorf("wrong program format")
	}
	v, err := vm.AsInt64(insts[2].Data)
	if err != nil {
		return nil, err
	}
	if v != 0 {
		return nil, fmt.Errorf("wrong program format")
	}
	if insts[3].Op != vm.OP_CHECKPREDICATE {
		return nil, fmt.Errorf("wrong program format")
	}
	return result, nil
}
