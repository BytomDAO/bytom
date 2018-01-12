package transaction

import (
	"github.com/bytom/consensus"
	"github.com/bytom/protocol/vm"
)

func IsP2WPKHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	return len(insts) == 1 && insts[0].Op == vm.OP_DATA_20 && len(insts[0].Data) == consensus.PayToWitnessPubKeyHashDataSize
}

func IsP2WSHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	return len(insts) == 1 && insts[0].Op == vm.OP_DATA_32 && len(insts[0].Data) == consensus.PayToWitnessScriptHashDataSize
}
