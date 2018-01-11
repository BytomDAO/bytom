package transaction

import (
	"github.com/bytom/crypto"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm"
)

var standardP2PKH = []vm.Op{vm.OP_DUP, vm.OP_HASH160, vm.OP_DATA_1, vm.OP_EQUALVERIFY, vm.OP_TXSIGHASH, vm.OP_SWAP, vm.OP_CHECKSIG}
var standardP2SH = []vm.Op{vm.OP_DUP, vm.OP_SHA3, vm.OP_DATA_1, vm.OP_EQUALVERIFY, vm.OP_DATA_1, vm.OP_SWAP, vm.OP_0, vm.OP_CHECKPREDICATE}

func IsStandardTxInput(in *legacy.TxInput) bool {
	spend, ok := in.TypedInput.(*legacy.SpendInput)
	if !ok {
		return false
	}
	if spend.ControlProgram[1] == vm.OP_HASH160 {
		return isP2PKH(spend.ControlProgram)
	}
	return isP2SH(spend.ControlProgram, spend.Arguments)
}

func isP2PKH(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	for _, op := range standardP2PKH {
	}
	return true
}

func isP2SH(program []byte, args [][]byte) bool {
	scriptHash := crypto.Sha256(args[1])
	return true
}

func IsStandardTxOutput(out *legacy.TxOutput) bool {
	return true
}
