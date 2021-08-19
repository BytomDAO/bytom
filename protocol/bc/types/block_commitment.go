package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
)

// BlockCommitment store the TransactionsMerkleRoot
type BlockCommitment struct {
	// TransactionsMerkleRoot is the root hash of the Merkle binary hash tree
	// formed by the hashes of all transactions included in the block.
	TransactionsMerkleRoot bc.Hash `json:"transaction_merkle_root"`
}

func (bc *BlockCommitment) readFrom(r *blockchain.Reader) error {
	_, err := bc.TransactionsMerkleRoot.ReadFrom(r)
	return err
}

func (bc *BlockCommitment) writeTo(w io.Writer) error {
	_, err := bc.TransactionsMerkleRoot.WriteTo(w)
	return err
}
