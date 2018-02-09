package instance

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/bytom/encoding/json"
	"github.com/bytom/exp/ivy/compiler"
	"github.com/bytom/protocol/vm"
)

// LockWithPublicKeyHashBodyBytes refer to contract's body
var LockWithPublicKeyHashBodyBytes []byte

func init() {
	LockWithPublicKeyHashBodyBytes, _ = hex.DecodeString("5279aa887cae7cac")
}

// contract LockWithPublicKeyHash(pubKeyHash: Hash) locks value
//
// 2                        [... pubKey sig pubKeyHash 2]
// PICK                     [... pubKey sig pubKeyHash pubKey]
// SHA3                     [... pubKey sig pubKeyHash sha3(pubKey)]
// SWAP                     [... pubKey sig sha3(pubKey) pubKeyHash]
// EQUAL                    [... pubKey sig (sha3(pubKey) == pubKeyHash)]
// VERIFY                   [... pubKey sig]
// SWAP                     [... sig pubKey]
// TXSIGHASH SWAP CHECKSIG  [... checkTxSig(pubKey, sig)]

// PayToLockWithPublicKeyHash instantiates contract LockWithPublicKeyHash as a program with specific arguments.
func PayToLockWithPublicKeyHash(pubKeyHash []byte) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "pubKeyHash", Type: "Hash"},
	}
	var _contractArgs []compiler.ContractArg
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&pubKeyHash)})
	return compiler.Instantiate(LockWithPublicKeyHashBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToLockWithPublicKeyHash parses the arguments out of an instantiation of contract LockWithPublicKeyHash.
// If the input is not an instantiation of LockWithPublicKeyHash, returns an error.
func ParsePayToLockWithPublicKeyHash(prog []byte) ([][]byte, error) {
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
	if !bytes.Equal(LockWithPublicKeyHashBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match LockWithPublicKeyHash")
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
