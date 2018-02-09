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

// LockWithMultiSigBodyBytes refer to contract's body
var LockWithMultiSigBodyBytes []byte

func init() {
	LockWithMultiSigBodyBytes, _ = hex.DecodeString("537a547a526bae71557a536c7cad")
}

// contract LockWithMultiSig(publicKey1: PublicKey, publicKey2: PublicKey, publicKey3: PublicKey) locks value
//
// 3              [... sig1 sig2 publicKey3 publicKey2 publicKey1 3]
// ROLL           [... sig1 publicKey3 publicKey2 publicKey1 sig2]
// 4              [... sig1 publicKey3 publicKey2 publicKey1 sig2 4]
// ROLL           [... publicKey3 publicKey2 publicKey1 sig2 sig1]
// 2              [... publicKey3 publicKey2 publicKey1 sig2 sig1 2]
// TOALTSTACK     [... publicKey3 publicKey2 publicKey1 sig2 sig1]
// TXSIGHASH      [... publicKey3 publicKey2 publicKey1 sig2 sig1 <txsighash>]
// 5              [... publicKey3 publicKey2 publicKey1 sig2 sig1 <txsighash> 5]
// ROLL           [... publicKey2 publicKey1 sig2 sig1 <txsighash> publicKey3]
// 5              [... publicKey2 publicKey1 sig2 sig1 <txsighash> publicKey3 5]
// ROLL           [... publicKey1 sig2 sig1 <txsighash> publicKey3 publicKey2]
// 5              [... publicKey1 sig2 sig1 <txsighash> publicKey3 publicKey2 5]
// ROLL           [... sig2 sig1 <txsighash> publicKey3 publicKey2 publicKey1]
// 3              [... sig2 sig1 <txsighash> publicKey3 publicKey2 publicKey1 3]
// FROMALTSTACK   [... sig2 sig1 <txsighash> publicKey3 publicKey2 publicKey1 3 2]
// SWAP           [... sig2 sig1 <txsighash> publicKey3 publicKey2 publicKey1 2 3]
// CHECKMULTISIG  [... sig2 checkTxMultiSig([publicKey1, publicKey2, publicKey3], [sig1, sig2])]

// PayToLockWithMultiSig instantiates contract LockWithMultiSig as a program with specific arguments.
func PayToLockWithMultiSig(publicKey1 ed25519.PublicKey, publicKey2 ed25519.PublicKey, publicKey3 ed25519.PublicKey) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "publicKey1", Type: "PublicKey"},
		{Name: "publicKey2", Type: "PublicKey"},
		{Name: "publicKey3", Type: "PublicKey"},
	}
	var _contractArgs []compiler.ContractArg
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&publicKey1)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&publicKey2)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&publicKey3)})
	return compiler.Instantiate(LockWithMultiSigBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToLockWithMultiSig parses the arguments out of an instantiation of contract LockWithMultiSig.
// If the input is not an instantiation of LockWithMultiSig, returns an error.
func ParsePayToLockWithMultiSig(prog []byte) ([][]byte, error) {
	var result [][]byte
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 3; i++ {
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
	if !bytes.Equal(LockWithMultiSigBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match LockWithMultiSig")
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
