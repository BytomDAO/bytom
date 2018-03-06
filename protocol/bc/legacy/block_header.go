package legacy

import (
	"fmt"
	"io"
	"time"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

// BlockHeader describes necessary data of the block.
type BlockHeader struct {
	// Version of the block.
	Version uint64 `json:"version"`

	// Height of the block in the block chain.
	// Initial block has height 1.
	Height uint64 `json:"height"`

	// Hash of the previous block in the block chain.
	PreviousBlockHash bc.Hash `json:"previous_block_hash"`

	// Time of the block in seconds.
	Timestamp uint64 `json:"timestamp"`

	BlockCommitment

	Nonce uint64 `json:"nonce"`
	Bits  uint64 `json:"bits"`
}

// Time returns the time represented by the Timestamp in bh.
func (bh *BlockHeader) Time() time.Time {
	tsNano := bh.Timestamp * uint64(time.Second)
	return time.Unix(0, int64(tsNano)).UTC()
}

// Scan validates the input byte slice is a valid block header
func (bh *BlockHeader) Scan(val interface{}) error {
	driverBuf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	buf := make([]byte, len(driverBuf))
	copy(buf[:], driverBuf)
	_, err := bh.readFrom(blockchain.NewReader(buf))
	return err
}

// Hash returns complete hash of the block header.
func (bh *BlockHeader) Hash() bc.Hash {
	h, _ := mapBlockHeader(bh)
	return h
}

func (bh *BlockHeader) readFrom(r *blockchain.Reader) (serflag uint8, err error) {
	var serflags [1]byte
	io.ReadFull(r, serflags[:])
	serflag = serflags[0]
	switch serflag {
	case SerBlockHeader, SerBlockFull:
	default:
		return 0, fmt.Errorf("unsupported serialization flags 0x%x", serflags)
	}

	if bh.Version, err = blockchain.ReadVarint63(r); err != nil {
		return 0, err
	}
	if bh.Height, err = blockchain.ReadVarint63(r); err != nil {
		return 0, err
	}
	if _, err = bh.PreviousBlockHash.ReadFrom(r); err != nil {
		return 0, err
	}
	if bh.Timestamp, err = blockchain.ReadVarint63(r); err != nil {
		return 0, err
	}
	if _, err = blockchain.ReadExtensibleString(r, bh.BlockCommitment.readFrom); err != nil {
		return 0, err
	}
	if bh.Nonce, err = blockchain.ReadVarint63(r); err != nil {
		return 0, err
	}
	if bh.Bits, err = blockchain.ReadVarint63(r); err != nil {
		return 0, err
	}
	return
}

// WriteTo writes the block header to the input io.Writer
func (bh *BlockHeader) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	bh.writeTo(ew, SerBlockHeader)
	return ew.Written(), ew.Err()
}

// writeTo writes bh to w.
func (bh *BlockHeader) writeTo(w io.Writer, serflags uint8) (err error) {
	w.Write([]byte{serflags})

	if _, err = blockchain.WriteVarint63(w, bh.Version); err != nil {
		return err
	}
	if _, err = blockchain.WriteVarint63(w, bh.Height); err != nil {
		return err
	}
	if _, err = bh.PreviousBlockHash.WriteTo(w); err != nil {
		return err
	}
	if _, err = blockchain.WriteVarint63(w, bh.Timestamp); err != nil {
		return err
	}
	if _, err = blockchain.WriteExtensibleString(w, nil, bh.BlockCommitment.writeTo); err != nil {
		return err
	}
	if _, err = blockchain.WriteVarint63(w, bh.Nonce); err != nil {
		return err
	}
	if _, err = blockchain.WriteVarint63(w, bh.Bits); err != nil {
		return err
	}
	return nil
}
