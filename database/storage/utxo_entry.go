package storage

const (
	NormalUTXOType uint32 = iota
	CoinbaseUTXOType
	VoteUTXOType
)

// NewUtxoEntry will create a new utxo entry
func NewUtxoEntry(utxoType uint32, blockHeight uint64, spent bool) *UtxoEntry {
	return &UtxoEntry{
		Type:        utxoType,
		BlockHeight: blockHeight,
		Spent:       spent,
	}
}

// SpendOutput marks the output at the provided index as spent
func (entry *UtxoEntry) SpendOutput() {
	entry.Spent = true
}

// UnspendOutput marks the output at the provided index as unspent
func (entry *UtxoEntry) UnspendOutput() {
	entry.Spent = false
}
