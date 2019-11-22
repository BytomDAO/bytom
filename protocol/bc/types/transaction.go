package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

const serRequired = 0x7 // Bit mask accepted serialization flag.

// Tx holds a transaction along with its hash.
type Tx struct {
	TxData
	*bc.Tx `json:"-"`
}

// NewTx returns a new Tx containing data and its hash. If you have already
// computed the hash, use struct literal notation to make a Tx object directly.
func NewTx(data TxData) *Tx {
	return &Tx{
		TxData: data,
		Tx:     MapTx(&data),
	}
}

// OutputID return the hash of the output position
func (tx *Tx) OutputID(outputIndex int) *bc.Hash {
	return tx.ResultIds[outputIndex]
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (tx *Tx) UnmarshalText(p []byte) error {
	if err := tx.TxData.UnmarshalText(p); err != nil {
		return err
	}

	tx.Tx = MapTx(&tx.TxData)
	return nil
}

// SetInputArguments sets the Arguments field in input n.
func (tx *Tx) SetInputArguments(n uint32, args [][]byte) {
	tx.Inputs[n].SetArguments(args)
	id := tx.Tx.InputIDs[n]
	e := tx.Entries[id]
	switch e := e.(type) {
	case *bc.Issuance:
		e.WitnessArguments = args
	case *bc.Spend:
		e.WitnessArguments = args
	}
}

// TxData encodes a transaction in the blockchain.
type TxData struct {
	Version        uint64
	SerializedSize uint64
	TimeRange      uint64
	Inputs         []*TxInput
	Outputs        []*TxOutput
}

// MarshalText fulfills the json.Marshaler interface.
func (tx *TxData) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := tx.WriteTo(&buf); err != nil {
		return nil, err
	}

	b := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(b, buf.Bytes())
	return b, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (tx *TxData) UnmarshalText(p []byte) error {
	b := make([]byte, hex.DecodedLen(len(p)))
	if _, err := hex.Decode(b, p); err != nil {
		return err
	}

	r := blockchain.NewReader(b)
	if err := tx.readFrom(r); err != nil {
		return err
	}

	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
}

func (tx *TxData) readFrom(r *blockchain.Reader) (err error) {
	startSerializedSize := r.Len()
	var serflags [1]byte
	if _, err = io.ReadFull(r, serflags[:]); err != nil {
		return errors.Wrap(err, "reading serialization flags")
	}
	if serflags[0] != serRequired {
		return fmt.Errorf("unsupported serflags %#x", serflags[0])
	}

	if tx.Version, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading transaction version")
	}
	if tx.TimeRange, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}

	n, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of transaction inputs")
	}

	for ; n > 0; n-- {
		ti := new(TxInput)
		if err = ti.readFrom(r); err != nil {
			return errors.Wrapf(err, "reading input %d", len(tx.Inputs))
		}
		tx.Inputs = append(tx.Inputs, ti)
	}

	n, err = blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of transaction outputs")
	}

	for ; n > 0; n-- {
		to := new(TxOutput)
		if err = to.readFrom(r); err != nil {
			return errors.Wrapf(err, "reading output %d", len(tx.Outputs))
		}
		tx.Outputs = append(tx.Outputs, to)
	}
	tx.SerializedSize = uint64(startSerializedSize - r.Len())
	return nil
}

// WriteTo writes tx to w.
func (tx *TxData) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	if err := tx.writeTo(ew, serRequired); err != nil {
		return 0, err
	}
	return ew.Written(), ew.Err()
}

func (tx *TxData) writeTo(w io.Writer, serflags byte) error {
	if _, err := w.Write([]byte{serflags}); err != nil {
		return errors.Wrap(err, "writing serialization flags")
	}
	if _, err := blockchain.WriteVarint63(w, tx.Version); err != nil {
		return errors.Wrap(err, "writing transaction version")
	}
	if _, err := blockchain.WriteVarint63(w, tx.TimeRange); err != nil {
		return errors.Wrap(err, "writing transaction maxtime")
	}

	if _, err := blockchain.WriteVarint31(w, uint64(len(tx.Inputs))); err != nil {
		return errors.Wrap(err, "writing tx input count")
	}

	for i, ti := range tx.Inputs {
		if err := ti.writeTo(w); err != nil {
			return errors.Wrapf(err, "writing tx input %d", i)
		}
	}

	if _, err := blockchain.WriteVarint31(w, uint64(len(tx.Outputs))); err != nil {
		return errors.Wrap(err, "writing tx output count")
	}

	for i, to := range tx.Outputs {
		if err := to.writeTo(w); err != nil {
			return errors.Wrapf(err, "writing tx output %d", i)
		}
	}
	return nil
}
