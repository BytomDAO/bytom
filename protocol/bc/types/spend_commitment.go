package types

import (
	"fmt"
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

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
		if _, err = sc.SourceID.WriteTo(w); err != nil {
			return errors.Wrap(err, "writing source id")
		}
		if _, err = sc.AssetAmount.WriteTo(w); err != nil {
			return errors.Wrap(err, "writing asset amount")
		}
		if _, err = blockchain.WriteVarint63(w, sc.SourcePosition); err != nil {
			return errors.Wrap(err, "writing source position")
		}
		if _, err = blockchain.WriteVarint63(w, sc.VMVersion); err != nil {
			return errors.Wrap(err, "writing vm version")
		}
		if _, err = blockchain.WriteVarstr31(w, sc.ControlProgram); err != nil {
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
			if _, err := sc.SourceID.ReadFrom(r); err != nil {
				return errors.Wrap(err, "reading source id")
			}
			if err = sc.AssetAmount.ReadFrom(r); err != nil {
				return errors.Wrap(err, "reading asset+amount")
			}
			if sc.SourcePosition, err = blockchain.ReadVarint63(r); err != nil {
				return errors.Wrap(err, "reading source position")
			}
			if sc.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
				return errors.Wrap(err, "reading VM version")
			}
			if sc.VMVersion != 1 {
				return fmt.Errorf("unrecognized VM version %d for asset version 1", sc.VMVersion)
			}
			if sc.ControlProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return errors.Wrap(err, "reading control program")
			}
			return nil
		}
		return nil
	})
}
