package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
)

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	SpendCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments             [][]byte // Witness
	SpendCommitment
}

// NewSpendInput create a new SpendInput struct.
func NewSpendInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte, stateData [][]byte) *TxInput {
	sc := SpendCommitment{
		AssetAmount: bc.AssetAmount{
			AssetId: &assetID,
			Amount:  amount,
		},
		SourceID:       sourceID,
		SourcePosition: sourcePos,
		VMVersion:      1,
		ControlProgram: controlProgram,
		StateData:      stateData,
	}
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &SpendInput{
			SpendCommitment: sc,
			Arguments:       arguments,
		},
	}
}

// AssetID implement the TypedInput.
func (si *SpendInput) AssetID() bc.AssetID {
	return *si.AssetId
}

// InputType is the interface function for return the input type.
func (si *SpendInput) InputType() uint8 { return SpendInputType }

func (si *SpendInput) readCommitment(r *blockchain.Reader) (err error) {
	si.SpendCommitmentSuffix, err = si.SpendCommitment.readFrom(r, 1)
	return err
}

func (si *SpendInput) readWitness(r *blockchain.Reader) (err error) {
	si.Arguments, err = blockchain.ReadVarstrList(r)
	return err
}

func (si *SpendInput) writeCommitment(w io.Writer, assetVersion uint64) error {
	if _, err := w.Write([]byte{SpendInputType}); err != nil {
		return err
	}

	return si.SpendCommitment.writeExtensibleString(w, si.SpendCommitmentSuffix, assetVersion)
}

func (si *SpendInput) writeWitness(w io.Writer) error {
	_, err := blockchain.WriteVarstrList(w, si.Arguments)
	return err
}
