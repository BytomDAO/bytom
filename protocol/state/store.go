package state

import (
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

// Store provides storage interface for blockchain data
type Store interface {
	BlockExist(*bc.Hash) bool

	GetBlock(*bc.Hash) (*types.Block, error)
	GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)
	GetStoreStatus() *BlockStoreState
	GetTransactionsUtxo(*UtxoViewpoint, []*bc.Tx) error
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)
	GetMainChainHash(uint64) (*bc.Hash, error)
	GetContract(hash [32]byte) ([]byte, error)

	GetCheckpoint(*bc.Hash) (*Checkpoint, error)
	CheckpointsFromNode(height uint64, hash *bc.Hash) ([]*Checkpoint, error)
	GetCheckpointsByHeight(uint64) ([]*Checkpoint, error)
	SaveCheckpoints([]*Checkpoint) error

	SaveBlock(*types.Block) error
	SaveBlockHeader(*types.BlockHeader) error
	SaveChainStatus(*types.BlockHeader, []*types.BlockHeader, *UtxoViewpoint, *ContractViewpoint, uint64, *bc.Hash) error
}

// BlockStoreState represents the core's db status
type BlockStoreState struct {
	Height          uint64
	Hash            *bc.Hash
	FinalizedHeight uint64
	FinalizedHash   *bc.Hash
}
