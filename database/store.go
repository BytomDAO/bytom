package database

import (
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

// Store provides storage interface for blockchain data
type Store interface {
	BlockExist(*bc.Hash) bool

	GetBlock(*bc.Hash) (*types.Block, error)
	GetMainchain(*bc.Hash) (map[uint64]*bc.Hash, error)
	GetStoreStatus() BlockStoreStateJSON
	GetSeed(*bc.Hash) (*bc.Hash, error)
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)

	SaveBlock(*types.Block, *bc.TransactionStatus, *bc.Hash) error
	SaveChainStatus(*types.Block, *state.UtxoViewpoint, map[uint64]*bc.Hash) error
}

// BlockStoreStateJSON represents the core's db status
type BlockStoreStateJSON struct {
	Height uint64
	Hash   *bc.Hash
}
