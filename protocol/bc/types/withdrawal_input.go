package types

import (
	"github.com/bytom/bytom/protocol/bc"
)

// WithdrawalInput satisfies the TypedInput interface and represents a veto transaction.
type WithdrawalInput struct {
	CommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments        [][]byte // Witness
	PubKey           []byte
	SpendCommitment
}

// NewWithdrawalInput create a new VetoInput struct.
func NewWithdrawalInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte, pubKey []byte) *TxInput {
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
		TypedInput: &WithdrawalInput{
			SpendCommitment: sc,
			Arguments:       arguments,
			PubKey:          pubKey,
		},
	}
}

// InputType is the interface function for return the input type.
func (w *WithdrawalInput) InputType() uint8 { return WithdrawalInputType }
