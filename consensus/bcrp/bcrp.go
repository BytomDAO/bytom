package bcrp

import (
	"bytes"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/vm"
)

// BCRP bytom contract register protocol
const BCRP = "bcrp"

// Version bcrp version
const Version = 1

// IsBCRPScript checks if a control program is bytom contract register protocol
// BCRP script format: OP_FAIL + OP_PUSHDATA1 + "04" + hex("bcrp") + OP_PUSHDATA1 + "01" + "01" + OP_PUSHDATA1 + len(contract) + contract
func IsBCRPScript(prog []byte) bool {
	inst, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	if len(inst) != 4 {
		return false
	}

	if inst[0].Op != vm.OP_FAIL {
		return false
	}

	if inst[1].Op != vm.OP_PUSHDATA1 {
		return false
	}

	if !bytes.Equal(inst[1].Data, []byte(BCRP)) {
		return false
	}

	if inst[2].Op != vm.OP_PUSHDATA1 {
		return false
	}

	// version 1
	if !bytes.Equal(inst[2].Data, []byte{byte(Version)}) {
		return false
	}

	if inst[3].Op != vm.OP_PUSHDATA1 {
		return false
	}

	return true
}

// IsCallBCRPScript checks if a control program is call contract registered by bytom contract register protocol
// call BCRP script format: OP_1 + OP_DATA_32 + SHA3-256(contract)
func IsCallBCRPScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	if len(insts) != 2 {
		return false
	}

	if insts[0].Op != vm.OP_1 {
		return false
	}

	return insts[1].Op == vm.OP_DATA_32 && len(insts[1].Data) == consensus.BCRPContractHashDataSize
}

// ParseContractHash parse contract hash from call BCRP script
// call BCRP script format: OP_1 + OP_DATA_32 + SHA3-256(contract)
func ParseContractHash(prog []byte) ([32]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return [32]byte{}, err
	}

	if len(insts) != 2 {
		return [32]byte{}, errors.New("unsupport program")
	}

	var hash [32]byte
	copy(hash[:], insts[1].Data)

	return hash, nil
}
