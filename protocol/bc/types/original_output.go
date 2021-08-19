package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
)

type originalTxOutput struct{}

func (o *originalTxOutput) OutputType() uint8                 { return OriginalOutputType }
func (o *originalTxOutput) readFrom(*blockchain.Reader) error { return nil }
func (o *originalTxOutput) writeTo(io.Writer) error           { return nil }

// NewOriginalTxOutput create a new original output struct
func NewOriginalTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte, stateData [][]byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
			StateData:      stateData,
		},
		TypedOutput: &originalTxOutput{},
	}
}
