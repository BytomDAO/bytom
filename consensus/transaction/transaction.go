package transaction

import (
	"github.com/bytom/protocol/bc/legacy"
)

func IsStandardTxInput(in *legacy.TxInput) bool {
	spend, ok := in.TypedInput.(*legacy.SpendInput)
	if !ok {
		return false
	}

	return true
}

func IsStandardTxOutput(out *legacy.TxOutput) bool {
	return true
}
