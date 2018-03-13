package legacy

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

// CurrentTransactionVersion is the current latest
// supported transaction version.
const CurrentTransactionVersion = 1

// Tx holds a transaction along with its hash.
type Tx struct {
	TxData
	*bc.Tx `json:"-"`
}

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

func (tx *Tx) IssuanceHash(n int) bc.Hash {
	return tx.Tx.InputIDs[n]
}

func (tx *Tx) OutputID(outputIndex int) *bc.Hash {
	return tx.ResultIds[outputIndex]
}

// NewTx returns a new Tx containing data and its hash.
// If you have already computed the hash, use struct literal
// notation to make a Tx object directly.
func NewTx(data TxData) *Tx {
	return &Tx{
		TxData: data,
		Tx:     MapTx(&data),
	}
}

// These flags are part of the wire protocol;
// they must not change.
const (
	SerWitness uint8 = 1 << iota
	SerPrevout
	SerMetadata

	// Bit mask for accepted serialization flags.
	// All other flag bits must be 0.
	SerTxHash   = 0x0 // this is used only for computing transaction hash - prevout and refdata are replaced with their hashes
	SerValid    = 0x7
	serRequired = 0x7 // we support only this combination of flags
)

// TxData encodes a transaction in the blockchain.
// Most users will want to use Tx instead;
// it includes the hash.
type TxData struct {
	Version        uint64
	SerializedSize uint64
	Inputs         []*TxInput
	Outputs        []*TxOutput

	TimeRange uint64

	// The unconsumed suffix of the common fields extensible string
	CommonFieldsSuffix []byte

	// The unconsumed suffix of the common witness extensible string
	CommonWitnessSuffix []byte
}

// HasIssuance returns true if this transaction has an issuance input.
func (tx *TxData) HasIssuance() bool {
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			return true
		}
	}
	return false
}

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
	tx.SerializedSize = uint64(r.Len())
	var serflags [1]byte
	if _, err = io.ReadFull(r, serflags[:]); err != nil {
		return errors.Wrap(err, "reading serialization flags")
	}
	if serflags[0] != serRequired {
		return fmt.Errorf("unsupported serflags %#x", serflags[0])
	}

	tx.Version, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading transaction version")
	}

	if tx.TimeRange, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}
	// Common witness
	tx.CommonWitnessSuffix, err = blockchain.ReadExtensibleString(r, tx.readCommonWitness)
	if err != nil {
		return errors.Wrap(err, "reading transaction common witness")
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
		if err = to.readFrom(r, tx.Version); err != nil {
			return errors.Wrapf(err, "reading output %d", len(tx.Outputs))
		}
		tx.Outputs = append(tx.Outputs, to)
	}

	return nil
}

// does not read the enclosing extensible string
func (tx *TxData) readCommonWitness(r *blockchain.Reader) error {
	return nil
}

func (tx *TxData) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	tx.WriteTo(&buf) // error is impossible
	b := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(b, buf.Bytes())
	return b, nil
}

// WriteTo writes tx to w.
func (tx *TxData) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	if err := tx.writeTo(ew, serRequired); err != nil {
		return ew.Written(), ew.Err()
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
	// common witness
	if _, err := blockchain.WriteExtensibleString(w, tx.CommonWitnessSuffix, tx.writeCommonWitness); err != nil {
		return errors.Wrap(err, "writing common witness")
	}

	if _, err := blockchain.WriteVarint31(w, uint64(len(tx.Inputs))); err != nil {
		return errors.Wrap(err, "writing tx input count")
	}
	for i, ti := range tx.Inputs {
		if err := ti.writeTo(w, serflags); err != nil {
			return errors.Wrapf(err, "writing tx input %d", i)
		}
	}

	if _, err := blockchain.WriteVarint31(w, uint64(len(tx.Outputs))); err != nil {
		return errors.Wrap(err, "writing tx output count")
	}
	for i, to := range tx.Outputs {
		if err := to.writeTo(w, serflags); err != nil {
			return errors.Wrapf(err, "writing tx output %d", i)
		}
	}

	return nil
}

// does not write the enclosing extensible string
func (tx *TxData) writeCommonWitness(w io.Writer) error {
	// Future protocol versions may add fields here.
	return nil
}

func writeRefData(w io.Writer, data []byte, serflags byte) error {
	if serflags&SerMetadata != 0 {
		_, err := blockchain.WriteVarstr31(w, data)
		return err
	}
	return writeFastHash(w, data)
}

func writeFastHash(w io.Writer, d []byte) error {
	if len(d) == 0 {
		_, err := blockchain.WriteVarstr31(w, nil)
		return err
	}
	var h [32]byte
	sha3pool.Sum256(h[:], d)
	_, err := blockchain.WriteVarstr31(w, h[:])
	return err
}
