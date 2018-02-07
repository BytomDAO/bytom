package legacy

import (
	"io"

	"github.com/bytom/encoding/blockchain"
	"github.com/bytom/protocol/bc"
)

// BlockCommitment store the TransactionsMerkleRoot && AssetsMerkleRoot
type BlockCommitment struct {
	// TransactionsMerkleRoot is the root hash of the Merkle binary hash
	// tree formed by the hashes of all transactions included in the
	// block.
	TransactionsMerkleRoot bc.Hash `json:"transaction_merkle_root"`

	// AssetsMerkleRoot is the root hash of the Merkle Patricia Tree of
	// the set of unspent outputs with asset version 1 after applying
	// the block.
	AssetsMerkleRoot bc.Hash `json:"asset_merkle_root"`
}

func (bc *BlockCommitment) readFrom(r *blockchain.Reader) error {
	if _, err := bc.TransactionsMerkleRoot.ReadFrom(r); err != nil {
		return err
	}
	_, err := bc.AssetsMerkleRoot.ReadFrom(r)
	return err
}

func (bc *BlockCommitment) writeTo(w io.Writer) error {
	_, err := bc.TransactionsMerkleRoot.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = bc.AssetsMerkleRoot.WriteTo(w)
	if err != nil {
		return err
	}
	return err
}
