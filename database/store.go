package database

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/common"

	"github.com/bytom/bytom/consensus"
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
	// CheckpointPrefix represent the namespace of checkpoints in db
	CheckpointPrefix = []byte("CP:")
	// BlockStoreKey block store key
	BlockStoreKey = []byte("blockStore")
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
func (s *Store) SaveChainStatus(node *state.BlockNode, view *state.UtxoViewpoint, contractView *state.ContractViewpoint, finalizedHeight uint64, finalizedHash *bc.Hash) error {
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

	bytes, err := json.Marshal(protocol.BlockStoreState{Height: node.Height, Hash: &node.Hash, FinalizedHeight: finalizedHeight, FinalizedHash: finalizedHash})
	if err != nil {
		return err
	}

	batch.Set(BlockStoreKey, bytes)
	batch.Write()
	return nil
}

func calcCheckpointKey(height uint64, hash *bc.Hash) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	key := append(CheckpointPrefix, buf...)
	if hash != nil {
		key = append(key, hash.Bytes()...)
	}
	return key
}

func (s *Store) GetCheckpoint(hash *bc.Hash) (*state.Checkpoint, error) {
	header, err := s.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	data := s.db.Get(calcCheckpointKey(header.Height, hash))
	checkpoint := &state.Checkpoint{}
	if err := json.Unmarshal(data, checkpoint); err != nil {
		return nil, err
	}

	setSupLinkToCheckpoint(checkpoint, header.SupLinks)
	return checkpoint, nil
}

// GetCheckpointsByHeight return all checkpoints of specified block height
func (s *Store) GetCheckpointsByHeight(height uint64) ([]*state.Checkpoint, error) {
	iter := s.db.IteratorPrefix(calcCheckpointKey(height, nil))
	defer iter.Release()
	return s.loadCheckpointsFromIter(iter)
}

// CheckpointsFromNode return all checkpoints from specified block height and hash
func (s *Store) CheckpointsFromNode(height uint64, hash *bc.Hash) ([]*state.Checkpoint, error) {
	startKey := calcCheckpointKey(height, hash)
	iter := s.db.IteratorPrefixWithStart(CheckpointPrefix, startKey, false)

	firstCheckpoint := &state.Checkpoint{}
	if err := json.Unmarshal(iter.Value(), firstCheckpoint); err != nil {
		return nil, err
	}

	checkpoints := []*state.Checkpoint{firstCheckpoint}
	subs, err := s.loadCheckpointsFromIter(iter)
	if err != nil {
		return nil, err
	}

	checkpoints = append(checkpoints, subs...)
	return checkpoints, nil
}

func (s *Store) loadCheckpointsFromIter(iter dbm.Iterator) ([]*state.Checkpoint, error) {
	var checkpoints []*state.Checkpoint
	defer iter.Release()
	for iter.Next() {
		checkpoint := &state.Checkpoint{}
		if err := json.Unmarshal(iter.Value(), checkpoint); err != nil {
			return nil, err
		}

		header, err := s.GetBlockHeader(&checkpoint.Hash)
		if err != nil {
			return nil, err
		}

		setSupLinkToCheckpoint(checkpoint, header.SupLinks)
		checkpoints = append(checkpoints, checkpoint)
	}
	return checkpoints, nil
}

// SaveCheckpoints bulk save multiple checkpoint
func (s *Store) SaveCheckpoints(checkpoints ...*state.Checkpoint) error {
	batch := s.db.NewBatch()
	for _, checkpoint := range checkpoints {
		data, err := json.Marshal(checkpoint)
		if err != nil {
			return err
		}

		if checkpoint.Height % state.BlocksOfEpoch != 1 {
			header, err := s.GetBlockHeader(&checkpoint.Hash)
			if err != nil {
				return err
			}

			batch.Delete(calcCheckpointKey(header.Height-1, &header.PreviousBlockHash))
		}

		batch.Set(calcCheckpointKey(checkpoint.Height, &checkpoint.Hash), data)
	}
	batch.Write()
	return nil
}

func setSupLinkToCheckpoint(c *state.Checkpoint, supLinks types.SupLinks) {
	for _, supLink := range supLinks {
		var signatures [consensus.MaxNumOfValidators]string
		for i, signature := range supLink.Signatures {
			signatures[i] = hex.EncodeToString(signature)
		}

		c.SupLinks = append(c.SupLinks, &state.SupLink{
			SourceHeight: supLink.SourceHeight,
			SourceHash:   supLink.SourceHash,
			Signatures:   signatures,
		})
	}
}
