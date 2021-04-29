package database

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/common"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

const logModule = "leveldb"

var (
	// BlockStoreKey block store key
	BlockStoreKey          = []byte("blockStore")
	// BlockHeaderIndexPrefix  block header index with height
	BlockHeaderIndexPrefix = []byte("BH:")
)

func loadBlockStoreStateJSON(db dbm.DB) *protocol.BlockStoreState {
	bytes := db.Get(BlockStoreKey)
	if bytes == nil {
		return nil
	}
	bsj := &protocol.BlockStoreState{}
	if err := json.Unmarshal(bytes, bsj); err != nil {
		common.PanicCrisis(common.Fmt("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type Store struct {
	db    dbm.DB
	cache cache
}

// NewStore creates and returns a new Store object.
func NewStore(db dbm.DB) *Store {
	fillBlockHeaderFn := func(hash *bc.Hash) (*types.BlockHeader, error) {
		return GetBlockHeader(db, hash)
	}

	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return GetBlockTransactions(db, hash)
	}

	fillBlockHashesFn := func(height uint64) ([]*bc.Hash, error) {
		return GetBlockHashesByHeight(db, height)
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillBlockHashesFn)
	return &Store{
		db:    db,
		cache: cache,
	}
}

// GetBlockHeader return the BlockHeader by given hash
func (s *Store) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.cache.lookupBlockHeader(hash)
}

// GetUtxo will search the utxo in db
func (s *Store) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return getUtxo(s.db, hash)
}

func (s *Store) GetContract(hash [32]byte) ([]byte, error) {
	return getContract(s.db, hash)
}

// BlockExist check if the block is stored in disk
func (s *Store) BlockExist(hash *bc.Hash) bool {
	_, err := s.cache.lookupBlockHeader(hash)
	return err == nil
}

// SaveBlockHeader persists a new block header in the protocol.
func (s *Store) SaveBlockHeader(blockHeader *types.BlockHeader) error {
	binaryBlockHeader, err := blockHeader.MarshalText()
	if err != nil {
		return errors.Wrap(err, "Marshal block header")
	}

	blockHash := blockHeader.Hash()
	s.db.Set(CalcBlockHeaderKey(&blockHash), binaryBlockHeader)
	s.cache.removeBlockHeader(blockHeader)
	return nil
}

// GetBlockHashesByHeight return the block hash by the specified height
func (s *Store) GetBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	return s.cache.lookupBlockHashesByHeight(height)
}

// SaveBlock persists a new block in the protocol.
func (s *Store) SaveBlock(block *types.Block) error {
	startTime := time.Now()
	binaryBlockHeader, err := block.MarshalTextForBlockHeader()
	if err != nil {
		return errors.Wrap(err, "Marshal block header")
	}

	binaryBlockTxs, err := block.MarshalTextForTransactions()
	if err != nil {
		return errors.Wrap(err, "Marshal block transactions")
	}

	blockHashes := []*bc.Hash{}
	hashes, err := s.GetBlockHashesByHeight(block.Height)
	if err != nil {
		return err
	}

	blockHashes = append(blockHashes, hashes...)
	blockHash := block.Hash()
	blockHashes = append(blockHashes, &blockHash)
	binaryBlockHashes, err := json.Marshal(blockHashes)
	if err != nil {
		return errors.Wrap(err, "Marshal block hashes")
	}

	batch := s.db.NewBatch()
	batch.Set(CalcBlockHashesKey(block.Height), binaryBlockHashes)
	batch.Set(CalcBlockHeaderKey(&blockHash), binaryBlockHeader)
	batch.Set(CalcBlockTransactionsKey(&blockHash), binaryBlockTxs)
	batch.Set(CalcBlockHeaderIndexKey(block.Height, &blockHash), binaryBlockHeader)
	batch.Write()

	s.cache.removeBlockHashes(block.Height)
	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   block.Height,
		"hash":     blockHash.String(),
		"duration": time.Since(startTime),
	}).Info("block saved on disk")
	return nil
}

// GetBlockTransactions return the Block transactions by given hash
func (s *Store) GetBlockTransactions(hash *bc.Hash) ([]*types.Tx, error) {
	return s.cache.lookupBlockTxs(hash)
}

// GetBlock return the block by given hash
func (s *Store) GetBlock(hash *bc.Hash) (*types.Block, error) {
	blockHeader, err := s.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	txs, err := s.GetBlockTransactions(hash)
	if err != nil {
		return nil, err
	}

	return &types.Block{
		BlockHeader:  *blockHeader,
		Transactions: txs,
	}, nil
}

// GetTransactionsUtxo will return all the utxo that related to the input txs
func (s *Store) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return getTransactionsUtxo(s.db, view, txs)
}

// GetStoreStatus return the BlockStoreStateJSON
func (s *Store) GetStoreStatus() *protocol.BlockStoreState {
	return loadBlockStoreStateJSON(s.db)
}

// LoadBlockIndex loadblockIndex by bestHeight
func (s *Store) LoadBlockIndex(stateBestHeight uint64) (*state.BlockIndex, error) {
	startTime := time.Now()
	blockIndex := state.NewBlockIndex()
	bhIter := s.db.IteratorPrefix(BlockHeaderIndexPrefix)
	defer bhIter.Release()

	var lastNode *state.BlockNode
	for bhIter.Next() {
		bh := &types.BlockHeader{}
		if err := bh.UnmarshalText(bhIter.Value()); err != nil {
			return nil, err
		}

		// If a block with a height greater than the best height of state is added to the index,
		// It may cause a bug that the new block cant not be process properly.
		if bh.Height > stateBestHeight {
			break
		}

		var parent *state.BlockNode
		if lastNode == nil || lastNode.Hash == bh.PreviousBlockHash {
			parent = lastNode
		} else {
			parent = blockIndex.GetNode(&bh.PreviousBlockHash)
		}

		node, err := state.NewBlockNode(bh, parent)
		if err != nil {
			return nil, err
		}

		blockIndex.AddNode(node)
		lastNode = node
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   stateBestHeight,
		"duration": time.Since(startTime),
	}).Debug("initialize load history block index from database")
	return blockIndex, nil
}

// SaveChainStatus save the core's newest status && delete old status
func (s *Store) SaveChainStatus(node *state.BlockNode, view *state.UtxoViewpoint, contractView *state.ContractViewpoint) error {
	batch := s.db.NewBatch()
	if err := saveUtxoView(batch, view); err != nil {
		return err
	}

	if err := deleteContractView(s.db, batch, contractView); err != nil {
		return err
	}

	if err := saveContractView(s.db, batch, contractView); err != nil {
		return err
	}

	bytes, err := json.Marshal(protocol.BlockStoreState{Height: node.Height, Hash: &node.Hash})
	if err != nil {
		return err
	}

	batch.Set(BlockStoreKey, bytes)
	batch.Write()
	return nil
}

func (s *Store) GetCheckpoint(*bc.Hash) (*state.Checkpoint, error) {
	return nil, nil
}

// GetCheckpointsByHeight return all checkpoints of specified block height
func (s *Store) GetCheckpointsByHeight(uint64) ([]*state.Checkpoint, error) {
	return nil, nil
}

// SaveCheckpoints bulk save multiple checkpoint
func (s *Store) SaveCheckpoints(...*state.Checkpoint) error {
	return nil
}
