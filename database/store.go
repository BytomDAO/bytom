package database

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

const logModule = "leveldb"

var (
	// BlockStoreKey block store key
	BlockStoreKey = []byte("blockStore")
)

func loadBlockStoreStateJSON(db dbm.DB) *state.BlockStoreState {
	bytes := db.Get(BlockStoreKey)
	if bytes == nil {
		return nil
	}
	bsj := &state.BlockStoreState{}
	if err := json.Unmarshal(bytes, bsj); err != nil {
		log.WithField("err", err).Panic("Could not unmarshal blockStoreState")
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

	fillMainChainHashFn := func(height uint64) (*bc.Hash, error) {
		return GetMainChainHash(db, height)
	}

	fillCheckPointFn := func(key []byte) (*state.Checkpoint, error) {
		return getCheckpointFromDB(db, key)
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillBlockHashesFn, fillMainChainHashFn, fillCheckPointFn)
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

// GetMainChainHash return the block hash by the specified height
func (s *Store) GetMainChainHash(height uint64) (*bc.Hash, error) {
	return s.cache.lookupMainChainHash(height)
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
func (s *Store) GetStoreStatus() *state.BlockStoreState {
	return loadBlockStoreStateJSON(s.db)
}

// SaveChainStatus save the core's newest status && delete old status
func (s *Store) SaveChainStatus(blockHeader *types.BlockHeader, mainBlockHeaders []*types.BlockHeader, view *state.UtxoViewpoint, contractView *state.ContractViewpoint, finalizedHeight uint64, finalizedHash *bc.Hash) error {
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

	blockHeaderHash := blockHeader.Hash()
	bytes, err := json.Marshal(
		state.BlockStoreState{
			Height:          blockHeader.Height,
			Hash:            &blockHeaderHash,
			FinalizedHeight: finalizedHeight,
			FinalizedHash:   finalizedHash,
		})
	if err != nil {
		return err
	}

	batch.Set(BlockStoreKey, bytes)

	var clearCacheFuncs []func()
	// save main chain blockHeaders
	for _, blockHeader := range mainBlockHeaders {
		bh := blockHeader
		blockHash := bh.Hash()
		binaryBlockHash, err := blockHash.MarshalText()
		if err != nil {
			return errors.Wrap(err, "Marshal block hash")
		}

		batch.Set(calcMainChainIndexPrefix(bh.Height), binaryBlockHash)
		clearCacheFuncs = append(clearCacheFuncs, func() {
			s.cache.removeMainChainHash(bh.Height)
		})
	}
	batch.Write()
	for _, clearCacheFunc := range clearCacheFuncs {
		clearCacheFunc()
	}

	return nil
}
