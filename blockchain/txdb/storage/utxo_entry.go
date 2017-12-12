package storage

func NewUtxoEntry(isCoinBase bool, blockHeight uint64) *UtxoEntry {
	return &UtxoEntry{
		IsCoinBase:  isCoinBase,
		BlockHeight: blockHeight,
		Spend:       false,
	}
}

// SpendOutput marks the output at the provided index as spent
func (entry *UtxoEntry) SpendOutput() {
	entry.Spend = true
}

// Unspend marks the output at the provided index as unspent
func (entry *UtxoEntry) UnspendOutput() {
	entry.Spend = false
}
