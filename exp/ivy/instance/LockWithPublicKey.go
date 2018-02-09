package instance

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/encoding/json"
	"github.com/bytom/exp/ivy/compiler"
	"github.com/bytom/protocol/vm"
)

// LockWithPublicKeyBodyBytes refer to contract's body
var LockWithPublicKeyBodyBytes []byte

func init() {
	LockWithPublicKeyBodyBytes, _ = hex.DecodeString("ae7cac")
}

// contract LockWithPublicKey(publicKey: PublicKey) locks locked
//
// SWAP                     [... publicKey sig]
// SWAP                     [... sig publicKey]
// TXSIGHASH SWAP CHECKSIG  [... checkTxSig(publicKey, sig)]

// PayToLockWithPublicKey instantiates contract LockWithPublicKey as a program with specific arguments.
func PayToLockWithPublicKey(publicKey ed25519.PublicKey) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "publicKey", Type: "PublicKey"},
	}
	var _contractArgs []compiler.ContractArg
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&publicKey)})
	return compiler.Instantiate(LockWithPublicKeyBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToLockWithPublicKey parses the arguments out of an instantiation of contract LockWithPublicKey.
// If the input is not an instantiation of LockWithPublicKey, returns an error.
func ParsePayToLockWithPublicKey(prog []byte) ([][]byte, error) {
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
	if !bytes.Equal(LockWithPublicKeyBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match LockWithPublicKey")
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
