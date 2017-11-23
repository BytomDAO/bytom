package txdb

import (
	"encoding/json"
	"fmt"

	"github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
)

var blockStoreKey = []byte("blockStore")

// BlockStoreStateJSON represents the core's db status
type BlockStoreStateJSON struct {
	Height uint64
	Hash   *bc.Hash
}

func (bsj BlockStoreStateJSON) save(db dbm.DB) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		common.PanicSanity(common.Fmt("Could not marshal state bytes: %v", err))
	}
	db.SetSync(blockStoreKey, bytes)
}

func loadBlockStoreStateJSON(db dbm.DB) BlockStoreStateJSON {
	bytes := db.Get(blockStoreKey)
	if bytes == nil {
		return BlockStoreStateJSON{
			Height: 0,
		}
	}
	bsj := BlockStoreStateJSON{}
	err := json.Unmarshal(bytes, &bsj)
	if err != nil {
		common.PanicCrisis(common.Fmt("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type Store struct {
	db    dbm.DB
	cache blockCache
}

func calcBlockKey(hash *bc.Hash) []byte {
	return []byte(fmt.Sprintf("B:%v", hash.String()))
}

// GetBlock return the block by given hash
func GetBlock(db dbm.DB, hash *bc.Hash) *legacy.Block {
	bytez := db.Get(calcBlockKey(hash))
	if bytez == nil {
		return nil
	}

	block := &legacy.Block{}
	block.UnmarshalText(bytez)
	return block
}

// NewStore creates and returns a new Store object.
func NewStore(db dbm.DB) *Store {
	cache := newBlockCache(func(hash *bc.Hash) *legacy.Block {
		return GetBlock(db, hash)
	})
	return &Store{
		db:    db,
		cache: cache,
	}
}

// BlockExist check if the block is stored in disk
func (s *Store) BlockExist(hash *bc.Hash) bool {
	block, err := s.cache.lookup(hash)
	return err == nil && block != nil
}

// GetBlock return the block by given hash
func (s *Store) GetBlock(hash *bc.Hash) (*legacy.Block, error) {
	return s.cache.lookup(hash)
}

// GetStoreStatus return the BlockStoreStateJSON
func (s *Store) GetStoreStatus() BlockStoreStateJSON {
	return loadBlockStoreStateJSON(s.db)
}

// GetMainchain read the mainchain map from db
func (s *Store) GetMainchain(hash *bc.Hash) (map[uint64]*bc.Hash, error) {
	return getMainchain(s.db, hash)
}

// GetSnapshot read the snapshot tree from db
func (s *Store) GetSnapshot(hash *bc.Hash) (*state.Snapshot, error) {
	return getSnapshot(s.db, hash)
}

// SaveBlock persists a new block in the database.
func (s *Store) SaveBlock(block *legacy.Block) error {
	binaryBlock, err := block.MarshalText()
	if err != nil {
		common.PanicCrisis(common.Fmt("Error Marshal block meta: %v", err))
	}

	blockHash := block.Hash()
	s.db.Set(calcBlockKey(&blockHash), binaryBlock)
	s.db.SetSync(nil, nil)
	return nil
}

// SaveMainchain saves the mainchain map to the database.
func (s *Store) SaveMainchain(mainchain map[uint64]*bc.Hash, hash *bc.Hash) error {
	err := saveMainchain(s.db, mainchain, hash)
	return errors.Wrap(err, "saving mainchain")
}

// SaveSnapshot saves a state snapshot to the database.
func (s *Store) SaveSnapshot(snapshot *state.Snapshot, hash *bc.Hash) error {
	err := saveSnapshot(s.db, snapshot, hash)
	return errors.Wrap(err, "saving state tree")
}

// SaveStoreStatus save the core's newest status && delete old status
func (s *Store) SaveStoreStatus(height uint64, hash *bc.Hash) {
	BlockStoreStateJSON{Height: height, Hash: hash}.save(s.db)
	cleanMainchainDB(s.db, hash)
	cleanSnapshotDB(s.db, hash)
}
