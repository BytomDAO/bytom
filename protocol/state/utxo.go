package state

import "github.com/bytom/protocol/bc"

// UtxoEntry contains contextual information about an unspent transaction
type UtxoEntry struct {
	modified    bool            // Entry changed since load.
	isCoinBase  bool            // Whether entry is a coinbase tx.
	blockHeight uint64          // Height of block containing tx.
	outputs     map[uint64]bool // Sparse map of unspent outputs.
}

// IsCoinBase returns whether or not it's a coinbase transaction
func (entry *UtxoEntry) IsCoinBase() bool {
	return entry.isCoinBase
}

// BlockHeight returns the height of the block containing the transaction
func (entry *UtxoEntry) BlockHeight() uint64 {
	return entry.blockHeight
}

// IsOutputSpent returns whether or not the output index has been spent
func (entry *UtxoEntry) IsOutputSpent(outputIndex uint64) bool {
	output, ok := entry.outputs[outputIndex]
	if !ok {
		return true
	}
	return output
}

// SpendOutput marks the output at the provided index as spent
func (entry *UtxoEntry) SpendOutput(outputIndex uint64) {
	output, ok := entry.outputs[outputIndex]
	if !ok || output {
		return
	}

	entry.modified = true
	entry.outputs[outputIndex] = true
}

// UtxoViewpoint represents a view into the set of unspent transaction outputs
type UtxoViewpoint struct {
	entries map[bc.Hash]*UtxoEntry
}
