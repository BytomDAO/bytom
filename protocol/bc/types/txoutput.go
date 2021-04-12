package types

import (
	"fmt"
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

const (
	// OriginalOutputType represent the type of original output
	OriginalOutputType uint8 = iota

	// VoteOutputType represent the type of vote output
	VoteOutputType
)

// TxOutput is the top level struct of tx output.
type TxOutput struct {
	AssetVersion uint64
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
	TypedOutput
}

// TypedOutput return the txoutput type.
type TypedOutput interface {
	OutputType() uint8
	readFrom(*blockchain.Reader) error
	writeTo(io.Writer) error
}

type originalTxOutput struct{}

func (o *originalTxOutput) OutputType() uint8                 { return OriginalOutputType }
func (o *originalTxOutput) readFrom(*blockchain.Reader) error { return nil }
func (o *originalTxOutput) writeTo(io.Writer) error           { return nil }

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
		TypedOutput: &originalTxOutput{},
	}
}

func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	var outType [1]byte
	if _, err = io.ReadFull(r, outType[:]); err != nil {
		return errors.Wrap(err, "reading output type")
	}

	var out TypedOutput
	switch outType[0] {
	case OriginalOutputType:
		out = new(originalTxOutput)
	case VoteOutputType:
		out = new(VoteOutput)
		to.TypedOutput = out
	default:
		return fmt.Errorf("unsupported output type %d", outType[0])
	}
	to.TypedOutput = out

	if to.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(reader *blockchain.Reader) error {
		if err := out.readFrom(reader); err != nil {
			return err
		}

		return to.OutputCommitment.readFrom(reader, to.AssetVersion)
	}); err != nil {
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

	if _, err := w.Write([]byte{to.OutputType()}); err != nil {
		return err
	}

	if _, err := blockchain.WriteExtensibleString(w, to.CommitmentSuffix, func(writer io.Writer) error {
		if err := to.TypedOutput.writeTo(writer); err != nil {
			return err
		}

		return to.OutputCommitment.writeTo(writer, to.AssetVersion)
	}); err != nil {
		return errors.Wrap(err, "writing output commitment")
	}

	if _, err := blockchain.WriteVarstr31(w, nil); err != nil {
		return errors.Wrap(err, "writing witness")
	}
	return nil
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
