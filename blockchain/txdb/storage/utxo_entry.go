package storage

// NewUtxoEntry will creaye a new utxo entry
func NewUtxoEntry(isCoinBase bool, blockHeight uint64, spend bool) *UtxoEntry {
	return &UtxoEntry{
		IsCoinBase:  isCoinBase,
		BlockHeight: blockHeight,
		Spend:       spend,
	}
}

// SpendOutput marks the output at the provided index as spent
func (entry *UtxoEntry) SpendOutput() {
	entry.Spend = true
}

// UnspendOutput marks the output at the provided index as unspent
func (entry *UtxoEntry) UnspendOutput() {
	entry.Spend = false
}
