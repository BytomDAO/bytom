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

// EscrowBodyBytes refer to contract's body
var EscrowBodyBytes []byte

func init() {
	EscrowBodyBytes, _ = hex.DecodeString("537a641b000000537a7cae7cac690000c3c251567ac1632a000000537a7cae7cac690000c3c251557ac1")
}

// contract Escrow(agent: PublicKey, sender: Program, recipient: Program) locks value
//
// 3                        [... <clause selector> recipient sender agent 3]
// ROLL                     [... recipient sender agent <clause selector>]
// JUMPIF:$reject           [... recipient sender agent]
// $approve                 [... recipient sender agent]
// 3                        [... sig recipient sender agent 3]
// ROLL                     [... recipient sender agent sig]
// SWAP                     [... recipient sender sig agent]
// TXSIGHASH SWAP CHECKSIG  [... recipient sender checkTxSig(agent, sig)]
// VERIFY                   [... recipient sender]
// 0                        [... recipient sender 0]
// 0                        [... recipient sender 0 0]
// AMOUNT                   [... recipient sender 0 0 <amount>]
// ASSET                    [... recipient sender 0 0 <amount> <asset>]
// 1                        [... recipient sender 0 0 <amount> <asset> 1]
// 6                        [... recipient sender 0 0 <amount> <asset> 1 6]
// ROLL                     [... sender 0 0 <amount> <asset> 1 recipient]
// CHECKOUTPUT              [... sender checkOutput(value, recipient)]
// JUMP:$_end               [... recipient sender agent]
// $reject                  [... recipient sender agent]
// 3                        [... sig recipient sender agent 3]
// ROLL                     [... recipient sender agent sig]
// SWAP                     [... recipient sender sig agent]
// TXSIGHASH SWAP CHECKSIG  [... recipient sender checkTxSig(agent, sig)]
// VERIFY                   [... recipient sender]
// 0                        [... recipient sender 0]
// 0                        [... recipient sender 0 0]
// AMOUNT                   [... recipient sender 0 0 <amount>]
// ASSET                    [... recipient sender 0 0 <amount> <asset>]
// 1                        [... recipient sender 0 0 <amount> <asset> 1]
// 5                        [... recipient sender 0 0 <amount> <asset> 1 5]
// ROLL                     [... recipient 0 0 <amount> <asset> 1 sender]
// CHECKOUTPUT              [... recipient checkOutput(value, sender)]
// $_end                    [... recipient sender agent]

// PayToEscrow instantiates contract Escrow as a program with specific arguments.
func PayToEscrow(agent ed25519.PublicKey, sender []byte, recipient []byte) ([]byte, error) {
	_contractParams := []*compiler.Param{
		{Name: "agent", Type: "PublicKey"},
		{Name: "sender", Type: "Program"},
		{Name: "recipient", Type: "Program"},
	}
	var _contractArgs []compiler.ContractArg
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&agent)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&sender)})
	_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&recipient)})
	return compiler.Instantiate(EscrowBodyBytes, _contractParams, false, _contractArgs)
}

// ParsePayToEscrow parses the arguments out of an instantiation of contract Escrow.
// If the input is not an instantiation of Escrow, returns an error.
func ParsePayToEscrow(prog []byte) ([][]byte, error) {
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
	if !bytes.Equal(EscrowBodyBytes, insts[1].Data) {
		return nil, fmt.Errorf("body bytes do not match Escrow")
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
