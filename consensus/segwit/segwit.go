package segwit

import (
	"bytes"
	"errors"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

//bcrp bytom contract register protocol
const bcrp = "bcrp"

func IsP2WScript(prog []byte) bool {
	return IsP2WPKHScript(prog) || IsP2WSHScript(prog) || IsStraightforward(prog)
}

func IsStraightforward(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 1 {
		return false
	}
	return insts[0].Op == vm.OP_TRUE || insts[0].Op == vm.OP_FAIL
}

func IsP2WPKHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_20 && len(insts[1].Data) == consensus.PayToWitnessPubKeyHashDataSize
}

func IsP2WSHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_32 && len(insts[1].Data) == consensus.PayToWitnessScriptHashDataSize
}

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

	if !bytes.Equal(inst[1].Data, []byte(bcrp)) {
		return false
	}

	if inst[2].Op != vm.OP_PUSHDATA1 {
		return false
	}

	// version 1
	if !bytes.Equal(inst[2].Data, []byte{byte(1)}) {
		return false
	}

	if inst[3].Op != vm.OP_PUSHDATA1 {
		return false
	}

	return true
}

func ConvertP2PKHSigProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2PKHSigProgram(insts[1].Data)
	}
	return nil, errors.New("unknow P2PKH version number")
}

func ConvertP2SHProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2SHProgram(insts[1].Data)
	}
	return nil, errors.New("unknow P2SHP version number")
}

func GetHashFromStandardProg(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[1].Data, nil
}
