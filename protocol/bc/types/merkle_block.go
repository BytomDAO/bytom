package types

// MerkleBlock proves the relevant transaction exists in a certain block through the Merkel tree.
type MerkleBlock struct {
	BlockHeader
	TransactionCount uint32
	TxHashes [][]byte
	TxFlags []byte
	StatusHashes [][]byte
	StatusFlags []byte
}