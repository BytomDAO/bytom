package bc

// Block is block struct in bc level
type Block struct {
	*BlockHeader
	ID           Hash
	Transactions []*Tx
}
