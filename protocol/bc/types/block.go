package types

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/encoding/bufpool"
	"github.com/bytom/errors"
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

// MarshalText fulfills the json.Marshaler interface. This guarantees that
// blocks will get deserialized correctly when being parsed from HTTP requests.
func (b *Block) MarshalText() ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	if _, err := b.WriteTo(buf); err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
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
	serflags, err := b.BlockHeader.readFrom(r)
	if err != nil {
		return err
	}

	if serflags == SerBlockHeader {
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

// WriteTo will write block to input io.Writer
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
