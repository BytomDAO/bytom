package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

// TxOutput is the top level struct of tx output.
type TxOutput struct {
	AssetVersion uint64
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
}

// NewTxOutput create a new output struct
func NewTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
		},
	}
}

func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	if to.CommitmentSuffix, err = to.OutputCommitment.readFrom(r, to.AssetVersion); err != nil {
		return errors.Wrap(err, "reading output commitment")
	}

	// read and ignore the (empty) output witness
	_, err = blockchain.ReadVarstr31(r)
	return errors.Wrap(err, "reading output witness")
}

func (to *TxOutput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, to.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	if err := to.writeCommitment(w); err != nil {
		return errors.Wrap(err, "writing output commitment")
	}

	if _, err := blockchain.WriteVarstr31(w, nil); err != nil {
		return errors.Wrap(err, "writing witness")
	}
	return nil
}

func (to *TxOutput) writeCommitment(w io.Writer) error {
	return to.OutputCommitment.writeExtensibleString(w, to.CommitmentSuffix, to.AssetVersion)
}

// ComputeOutputID assembles an output entry given a spend commitment and
// computes and returns its corresponding entry ID.
func ComputeOutputID(sc *SpendCommitment) (h bc.Hash, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()
	src := &bc.ValueSource{
		Ref:      &sc.SourceID,
		Value:    &sc.AssetAmount,
		Position: sc.SourcePosition,
	}
	o := bc.NewOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, 0)

	h = bc.EntryID(o)
	return h, nil
}
