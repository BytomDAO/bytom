package types

import (
	"fmt"
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

// serflag variables for input types.
const (
	IssuanceInputType uint8 = iota
	SpendInputType
	CoinbaseInputType
	VetoInputType
)

var inputTypeMap = map[uint8]func() TypedInput{
	IssuanceInputType: func() TypedInput { return &IssuanceInput{} },
	SpendInputType:    func() TypedInput { return &SpendInput{} },
	CoinbaseInputType: func() TypedInput { return &CoinbaseInput{} },
	VetoInputType:     func() TypedInput { return &VetoInput{} },
}

func parseTypedInput(r *blockchain.Reader) (TypedInput, error) {
	var inputType [1]byte
	if _, err := io.ReadFull(r, inputType[:]); err != nil {
		return nil, errors.Wrap(err, "reading input type")
	}

	newInFun, ok := inputTypeMap[inputType[0]]
	if !ok {
		return nil, fmt.Errorf("unsupported input type %d", inputType[0])
	}

	return newInFun(), nil
}

type (
	// TxInput is the top level struct of tx input.
	TxInput struct {
		AssetVersion uint64
		TypedInput
		CommitmentSuffix []byte
		WitnessSuffix    []byte
	}

	// TypedInput return the txinput type.
	TypedInput interface {
		InputType() uint8
		AssetID() bc.AssetID
		readCommitment(*blockchain.Reader) error
		readWitness(*blockchain.Reader) error
		writeCommitment(io.Writer, uint64) error
		writeWitness(w io.Writer) error
	}
)

// Amount return the asset amount of the txinput
func (t *TxInput) Amount() uint64 {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Amount
	case *SpendInput:
		return inp.Amount
	case *VetoInput:
		return inp.Amount
	}
	return 0
}

// Arguments get the args for the input
func (t *TxInput) Arguments() [][]byte {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Arguments
	case *SpendInput:
		return inp.Arguments
	case *VetoInput:
		return inp.Arguments
	}
	return nil
}

// ControlProgram return the control program of the spend input
func (t *TxInput) ControlProgram() []byte {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return inp.ControlProgram
	case *VetoInput:
		return inp.ControlProgram
	case *IssuanceInput:
		return inp.IssuanceProgram
	}
	return nil
}

// SetArguments set the args for the input
func (t *TxInput) SetArguments(args [][]byte) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		inp.Arguments = args
	case *SpendInput:
		inp.Arguments = args
	case *VetoInput:
		inp.Arguments = args
	}
}

// SpentOutputID calculate the hash of spended output
func (t *TxInput) SpentOutputID() (o bc.Hash, err error) {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		o, err = ComputeOutputID(&inp.SpendCommitment, inp.InputType(), nil)
	case *VetoInput:
		o, err = ComputeOutputID(&inp.SpendCommitment, inp.InputType(), inp.Vote)
	}
	return o, err
}

func (t *TxInput) readFrom(r *blockchain.Reader) (err error) {
	if t.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}

	t.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if t.AssetVersion != 1 {
			return nil
		}

		if t.TypedInput, err = parseTypedInput(r); err != nil {
			return err
		}

		return t.readCommitment(r)
	})
	if err != nil {
		return err
	}

	t.WitnessSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if t.AssetVersion == 1 {
			return t.readWitness(r)
		}

		return nil
	})

	return err
}

func (t *TxInput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, t.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	if _, err := blockchain.WriteExtensibleString(w, t.CommitmentSuffix, t.writeInputCommitment); err != nil {
		return errors.Wrap(err, "writing input commitment")
	}

	_, err := blockchain.WriteExtensibleString(w, t.WitnessSuffix, t.writeInputWitness)
	return errors.Wrap(err, "writing input witness")
}

func (t *TxInput) writeInputCommitment(w io.Writer) (err error) {
	if t.AssetVersion == 1 {
		return t.writeCommitment(w, t.AssetVersion)
	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) error {
	if t.AssetVersion == 1 {
		return t.writeWitness(w)
	}
	return nil
}
