package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

// GuarantyOutput satisfies the TypedOutput interface and represents a vote transaction.
type GuarantyOutput struct {
	PubKey []byte
}

// NewGuarantyOutput create a new output struct
func NewGuarantyOutput(assetID bc.AssetID, amount uint64, controlProgram []byte, pubKey []byte) *TxOutput {
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
		TypedOutput: &GuarantyOutput{PubKey: pubKey},
	}
}

func (g *GuarantyOutput) readFrom(r *blockchain.Reader) error {
	var err error
	if g.PubKey, err = blockchain.ReadVarstr31(r); err != nil {
		return errors.Wrap(err, "reading vote output vote")
	}
	return nil
}

func (g *GuarantyOutput) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstr31(w, g.PubKey)
	return err
}

// OutputType implement the txout interface
func (g *GuarantyOutput) OutputType() uint8 { return GuarantyOutputType }
