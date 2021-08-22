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

var outputTypeMap = map[uint8]func() TypedOutput{
	OriginalOutputType: func() TypedOutput { return &originalTxOutput{} },
	VoteOutputType:     func() TypedOutput { return &VoteOutput{} },
}

func parseTypedOutput(r *blockchain.Reader) (TypedOutput, error) {
	var outType [1]byte
	if _, err := io.ReadFull(r, outType[:]); err != nil {
		return nil, errors.Wrap(err, "reading output type")
	}

	newOutFun, ok := outputTypeMap[outType[0]]
	if !ok {
		return nil, fmt.Errorf("unsupported output type %d", outType[0])
	}

	return newOutFun(), nil
}

func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	to.TypedOutput, err = parseTypedOutput(r)
	if err != nil {
		return errors.Wrap(err, "parse typedOutput")
	}

	if to.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(reader *blockchain.Reader) error {
		if err := to.TypedOutput.readFrom(reader); err != nil {
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
func ComputeOutputID(sc *SpendCommitment, inputType uint8, vote []byte) (h bc.Hash, err error) {
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

	var o bc.Entry
	switch inputType {
	case SpendInputType:
		o = bc.NewOriginalOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, sc.StateData, 0)
	case VetoInputType:
		o = bc.NewVoteOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, sc.StateData, 0, vote)
	default:
		return h, fmt.Errorf("input type error:[%v]", inputType)
	}

	return bc.EntryID(o), nil
}
