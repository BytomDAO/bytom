package types

import (
	"fmt"
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

// OutputCommitment contains the commitment data for a transaction output.
type OutputCommitment struct {
	bc.AssetAmount
	VMVersion      uint64
	ControlProgram []byte
	StateData      [][]byte
}

func (oc *OutputCommitment) writeTo(w io.Writer, assetVersion uint64) (err error) {
	if assetVersion == 1 {
		if _, err = oc.AssetAmount.WriteTo(w); err != nil {
			return errors.Wrap(err, "writing asset amount")
		}
		if _, err = blockchain.WriteVarint63(w, oc.VMVersion); err != nil {
			return errors.Wrap(err, "writing vm version")
		}
		if _, err = blockchain.WriteVarstr31(w, oc.ControlProgram); err != nil {
			return errors.Wrap(err, "writing control program")
		}
		if _, err = blockchain.WriteVarstrList(w, oc.StateData); err != nil {
			return errors.Wrap(err, "writing state data")
		}
	}
	return errors.Wrap(err, "writing suffix")
}

func (oc *OutputCommitment) readFrom(r *blockchain.Reader, assetVersion uint64) (err error) {
	if assetVersion == 1 {
		if err := oc.AssetAmount.ReadFrom(r); err != nil {
			return errors.Wrap(err, "reading asset+amount")
		}
		oc.VMVersion, err = blockchain.ReadVarint63(r)
		if err != nil {
			return errors.Wrap(err, "reading VM version")
		}
		if oc.VMVersion != 1 {
			return fmt.Errorf("unrecognized VM version %d for asset version 1", oc.VMVersion)
		}
		oc.ControlProgram, err = blockchain.ReadVarstr31(r)
		if err != nil {
			return errors.Wrap(err, "reading control program")
		}
		oc.StateData, err = blockchain.ReadVarstrList(r)
		return errors.Wrap(err, "reading state data")
	}
	return nil
}
