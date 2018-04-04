package types

import (
	"fmt"
	"io"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	SpendCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments             [][]byte // Witness
	SpendCommitment
}

// NewSpendInput create a new SpendInput struct.
func NewSpendInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte) *TxInput {
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
		TypedInput: &SpendInput{
			SpendCommitment: sc,
			Arguments:       arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (si *SpendInput) InputType() uint8 { return SpendInputType }

// SpendCommitment contains the commitment data for a transaction output.
type SpendCommitment struct {
	bc.AssetAmount
	SourceID       bc.Hash
	SourcePosition uint64
	VMVersion      uint64
	ControlProgram []byte
}

func (sc *SpendCommitment) writeExtensibleString(w io.Writer, suffix []byte, assetVersion uint64) error {
	_, err := blockchain.WriteExtensibleString(w, suffix, func(w io.Writer) error {
		return sc.writeContents(w, suffix, assetVersion)
	})
	return err
}

func (sc *SpendCommitment) writeContents(w io.Writer, suffix []byte, assetVersion uint64) (err error) {
	if assetVersion == 1 {
		_, err = sc.SourceID.WriteTo(w)
		if err != nil {
			return errors.Wrap(err, "writing source id")
		}
		_, err = sc.AssetAmount.WriteTo(w)
		if err != nil {
			return errors.Wrap(err, "writing asset amount")
		}
		_, err = blockchain.WriteVarint63(w, sc.SourcePosition)
		if err != nil {
			return errors.Wrap(err, "writing source position")
		}
		_, err = blockchain.WriteVarint63(w, sc.VMVersion)
		if err != nil {
			return errors.Wrap(err, "writing vm version")
		}
		_, err = blockchain.WriteVarstr31(w, sc.ControlProgram)
		if err != nil {
			return errors.Wrap(err, "writing control program")
		}
	}
	if len(suffix) > 0 {
		_, err = w.Write(suffix)
	}
	return errors.Wrap(err, "writing suffix")
}

func (sc *SpendCommitment) readFrom(r *blockchain.Reader, assetVersion uint64) (suffix []byte, err error) {
	return blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if assetVersion == 1 {
			_, err := sc.SourceID.ReadFrom(r)
			if err != nil {
				return errors.Wrap(err, "reading source id")
			}
			err = sc.AssetAmount.ReadFrom(r)
			if err != nil {
				return errors.Wrap(err, "reading asset+amount")
			}
			sc.SourcePosition, err = blockchain.ReadVarint63(r)
			if err != nil {
				return errors.Wrap(err, "reading source position")
			}
			sc.VMVersion, err = blockchain.ReadVarint63(r)
			if err != nil {
				return errors.Wrap(err, "reading VM version")
			}
			if sc.VMVersion != 1 {
				return fmt.Errorf("unrecognized VM version %d for asset version 1", sc.VMVersion)
			}
			sc.ControlProgram, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return errors.Wrap(err, "reading control program")
			}
			return nil
		}
		return nil
	})
}
