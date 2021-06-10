package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
)

// VetoInput satisfies the TypedInput interface and represents a veto transaction.
type VetoInput struct {
	VetoCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments            [][]byte // Witness
	Vote                 []byte   // voter xpub
	SpendCommitment
}

// NewVetoInput create a new VetoInput struct.
func NewVetoInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte, vote []byte, stateData [][]byte) *TxInput {
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
		TypedInput: &VetoInput{
			SpendCommitment: sc,
			Arguments:       arguments,
			Vote:            vote,
		},
	}
}

// AssetID implement the TypedInput.
func (vi *VetoInput) AssetID() bc.AssetID {
	return *vi.AssetId
}

// InputType is the interface function for return the input type.
func (vi *VetoInput) InputType() uint8 { return VetoInputType }

func (vi *VetoInput) readCommitment(r *blockchain.Reader) (err error) {
	if vi.VetoCommitmentSuffix, err = vi.SpendCommitment.readFrom(r, 1); err != nil {
		return
	}

	vi.Vote, err = blockchain.ReadVarstr31(r)
	return
}

func (vi *VetoInput) readWitness(r *blockchain.Reader) (err error) {
	vi.Arguments, err = blockchain.ReadVarstrList(r)
	return err
}

func (vi *VetoInput) writeCommitment(w io.Writer, assetVersion uint64) error {
	if _, err := w.Write([]byte{VetoInputType}); err != nil {
		return err
	}

	if err := vi.SpendCommitment.writeExtensibleString(w, vi.VetoCommitmentSuffix, assetVersion); err != nil {
		return err
	}

	_, err := blockchain.WriteVarstr31(w, vi.Vote)
	return err
}

func (vi *VetoInput) writeWitness(w io.Writer) error {
	_, err := blockchain.WriteVarstrList(w, vi.Arguments)
	return err
}
