package txdb

import (
	"encoding/json"
	"fmt"

	. "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
)

var blockStoreKey = []byte("blockStore")

type BlockStoreStateJSON struct {
	Height uint64
	Hash   *bc.Hash
}

func (bsj BlockStoreStateJSON) Save(db dbm.DB) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		PanicSanity(Fmt("Could not marshal state bytes: %v", err))
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
		PanicCrisis(Fmt("Could not unmarshal bytes: %X", bytes))
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

func GetBlock(db dbm.DB, hash *bc.Hash) *legacy.Block {
	var block *legacy.Block = &legacy.Block{}
	bytez := db.Get(calcBlockKey(hash))
	if bytez == nil {
		return nil
	}

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

func (s *Store) BlockExist(hash *bc.Hash) bool {
	block, err := s.cache.lookup(hash)
	return err == nil && block != nil
}

func (s *Store) GetBlock(hash *bc.Hash) (*legacy.Block, error) {
	return s.cache.lookup(hash)
}

func (s *Store) GetStoreStatus() BlockStoreStateJSON {
	return loadBlockStoreStateJSON(s.db)
}

func (s *Store) GetMainchain() (map[uint64]*bc.Hash, MainchainStatusJSON, error) {
	return getMainchain(s.db)
}

func (s *Store) GetRawBlock(hash *bc.Hash) ([]byte, error) {
	bytez := s.db.Get(calcBlockKey(hash))
	if bytez == nil {
		return nil, errors.New("querying blocks from the db null")
	}
	return bytez, nil
}

func (s *Store) GetSnapshot() (*state.Snapshot, SnapshotStatusJSON, error) {
	return getSnapshot(s.db)
}

// SaveBlock persists a new block in the database.
func (s *Store) SaveBlock(block *legacy.Block) error {
	binaryBlock, err := block.MarshalText()
	if err != nil {
		PanicCrisis(Fmt("Error Marshal block meta: %v", err))
	}

	blockHash := block.Hash()
	s.db.Set(calcBlockKey(&blockHash), binaryBlock)
	s.db.SetSync(nil, nil)
	return nil
}

func (s *Store) SaveMainchain(mainchain map[uint64]*bc.Hash, height uint64, hash *bc.Hash) error {
	err := saveMainchain(s.db, mainchain, height, hash)
	return errors.Wrap(err, "saving mainchain")
}

// SaveSnapshot saves a state snapshot to the database.
func (s *Store) SaveSnapshot(snapshot *state.Snapshot, height uint64, hash *bc.Hash) error {
	err := saveSnapshot(s.db, snapshot, height, hash)
	return errors.Wrap(err, "saving state tree")
}

func (s *Store) SaveStoreStatus(height uint64, hash *bc.Hash) {
	BlockStoreStateJSON{Height: height, Hash: hash}.Save(s.db)
}
