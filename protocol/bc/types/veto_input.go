package types

import (
	"github.com/bytom/vapor/protocol/bc"
)

// VetoInput satisfies the TypedInput interface and represents a veto transaction.
type VetoInput struct {
	VetoCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments            [][]byte // Witness
	Vote                 []byte   // voter xpub
	SpendCommitment
}

// NewVetoInput create a new VetoInput struct.
func NewVetoInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte, vote []byte) *TxInput {
	sc := SpendCommitment{
		AssetAmount: bc.AssetAmount{
			AssetId: &assetID,
			Amount:  amount,
		},
		SourceID:       sourceID,
		SourcePosition: sourcePos,
		VMVersion:      1,
		ControlProgram: controlProgram,
	}
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &VetoInput{
			SpendCommitment: sc,
			Arguments:       arguments,
			Vote:            vote,
		},
	}
}

// InputType is the interface function for return the input type.
func (ui *VetoInput) InputType() uint8 { return VetoInputType }
