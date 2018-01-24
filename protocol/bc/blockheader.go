package bc

import "io"

// BlockHeader contains the header information for a blockchain
// block. It satisfies the Entry interface.

func (BlockHeader) typ() string { return "blockheader" }
func (bh *BlockHeader) writeForHash(w io.Writer) {
	mustWriteForHash(w, bh.Version)
	mustWriteForHash(w, bh.Height)
	mustWriteForHash(w, bh.PreviousBlockId)
	mustWriteForHash(w, bh.Seed)
	mustWriteForHash(w, bh.TimestampMs)
	mustWriteForHash(w, bh.TransactionsRoot)
	mustWriteForHash(w, bh.AssetsRoot)
	mustWriteForHash(w, bh.Nonce)
	mustWriteForHash(w, bh.Bits)
	mustWriteForHash(w, bh.TransactionStatus)
}

// NewBlockHeader creates a new BlockHeader and populates
// its body.
func NewBlockHeader(version, height uint64, previousBlockID, seed *Hash, timestampMS uint64, transactionsRoot, assetsRoot *Hash, ts *TransactionStatus, nonce, bits uint64) *BlockHeader {
	return &BlockHeader{
		Version:           version,
		Height:            height,
		PreviousBlockId:   previousBlockID,
		Seed:              seed,
		TimestampMs:       timestampMS,
		TransactionsRoot:  transactionsRoot,
		AssetsRoot:        assetsRoot,
		TransactionStatus: ts,
		Nonce:             nonce,
		Bits:              bits,
	}
}
