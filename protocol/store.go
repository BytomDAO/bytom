package protocol

import (
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

// Store provides storage interface for blockchain data
type Store interface {
	BlockExist(*bc.Hash) bool

	GetBlock(*bc.Hash) (*types.Block, error)
	GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)
	GetStoreStatus() *BlockStoreState
	GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)
	GetContract(hash [32]byte) ([]byte, error)

	GetCheckpoint(*bc.Hash) (*state.Checkpoint, error)
	CheckpointsFromNode(height uint64, hash *bc.Hash) ([]*state.Checkpoint, error)
	GetCheckpointsByHeight(uint64) ([]*state.Checkpoint, error)
	SaveCheckpoints(...*state.Checkpoint) error

	LoadBlockIndex(uint64) (*state.BlockIndex, error)
	SaveBlock(*types.Block) error
	SaveBlockHeader(*types.BlockHeader) error
	SaveChainStatus(*state.BlockNode, *state.UtxoViewpoint, *state.ContractViewpoint, uint64, *bc.Hash) error
}

// BlockStoreState represents the core's db status
type BlockStoreState struct {
	Height          uint64
	Hash            *bc.Hash
	FinalizedHeight uint64
	FinalizedHash   *bc.Hash
}
