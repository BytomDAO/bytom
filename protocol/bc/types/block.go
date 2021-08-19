package types

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/encoding/bufpool"
	"github.com/bytom/bytom/errors"
)

// serflag variables, start with 1
const (
	_ = iota
	SerBlockHeader
	SerBlockTransactions
	SerBlockFull
)

// Block describes a complete block, including its header and the transactions
// it contains.
type Block struct {
	BlockHeader
	Transactions []*Tx
}

func (b *Block) marshalText(serflags uint8) ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	ew := errors.NewWriter(buf)
	if err := b.writeTo(ew, serflags); err != nil {
		return nil, err
	}

	if err := ew.Err(); err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

// MarshalText fulfills the json.Marshaler interface. This guarantees that
// blocks will get deserialized correctly when being parsed from HTTP requests.
func (b *Block) MarshalText() ([]byte, error) {
	return b.marshalText(SerBlockFull)
}

// MarshalTextForBlockHeader fulfills the json.Marshaler interface.
func (b *Block) MarshalTextForBlockHeader() ([]byte, error) {
	return b.marshalText(SerBlockHeader)
}

// MarshalTextForTransactions fulfills the json.Marshaler interface.
func (b *Block) MarshalTextForTransactions() ([]byte, error) {
	return b.marshalText(SerBlockTransactions)
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (b *Block) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	if _, err := hex.Decode(decoded, text); err != nil {
		return err
	}

	r := blockchain.NewReader(decoded)
	if err := b.readFrom(r); err != nil {
		return err
	}

	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
}

func (b *Block) readFrom(r *blockchain.Reader) error {
	serflag, err := b.BlockHeader.readFrom(r)
	if err != nil {
		return err
	}

	if serflag == SerBlockHeader {
		return nil
	}

	n, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of transactions")
	}

	for ; n > 0; n-- {
		data := TxData{}
		if err = data.readFrom(r); err != nil {
			return errors.Wrapf(err, "reading transaction %d", len(b.Transactions))
		}

		b.Transactions = append(b.Transactions, NewTx(data))
	}
	return nil
}

// WriteTo write block to io.Writer
func (b *Block) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	if err := b.writeTo(ew, SerBlockFull); err != nil {
		return 0, err
	}
	return ew.Written(), ew.Err()
}

func (b *Block) writeTo(w io.Writer, serflags uint8) error {
	if err := b.BlockHeader.writeTo(w, serflags); err != nil {
		return err
	}

	if serflags == SerBlockHeader {
		return nil
	}

	if _, err := blockchain.WriteVarint31(w, uint64(len(b.Transactions))); err != nil {
		return err
	}

	for _, tx := range b.Transactions {
		if _, err := tx.WriteTo(w); err != nil {
			return err
		}
	}
	return nil
}
