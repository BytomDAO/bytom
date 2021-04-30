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
	GetCheckpointsByHeight(uint64) ([]*state.Checkpoint, error)
	SaveCheckpoints(...*state.Checkpoint) error

	LoadBlockIndex(uint64) (*state.BlockIndex, error)
	SaveBlock(*types.Block) error
	SaveChainStatus(*state.BlockNode, *state.UtxoViewpoint, *state.ContractViewpoint, *BlockStoreState) error
}
