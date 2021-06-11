package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
)

// CoinbaseInput records the coinbase message
type CoinbaseInput struct {
	Arbitrary []byte
}

// NewCoinbaseInput creates a new coinbase input struct
func NewCoinbaseInput(arbitrary []byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput:   &CoinbaseInput{Arbitrary: arbitrary},
	}
}

// AssetID implement the TypedInput.
func (cb *CoinbaseInput) AssetID() bc.AssetID {
	return bc.AssetID{}
}

// InputType is the interface function for return the input type
func (cb *CoinbaseInput) InputType() uint8 { return CoinbaseInputType }

func (cb *CoinbaseInput) readCommitment(r *blockchain.Reader) (err error) {
	cb.Arbitrary, err = blockchain.ReadVarstr31(r)
	return err
}

func (cb *CoinbaseInput) readWitness(_ *blockchain.Reader) error { return nil }

func (cb *CoinbaseInput) writeCommitment(w io.Writer, _ uint64) error {
	if _, err := w.Write([]byte{CoinbaseInputType}); err != nil {
		return err
	}

	_, err := blockchain.WriteVarstr31(w, cb.Arbitrary)
	return err
}

func (cb *CoinbaseInput) writeWitness(_ io.Writer) error { return nil }
