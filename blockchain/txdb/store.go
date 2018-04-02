package txdb

import (
	"encoding/json"
	"fmt"

	"github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/storage"
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

func (bsj BlockStoreStateJSON) save(batch dbm.Batch) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		common.PanicSanity(common.Fmt("Could not marshal state bytes: %v", err))
	}
	batch.Set(blockStoreKey, bytes)
}

func loadBlockStoreStateJSON(db dbm.DB) BlockStoreStateJSON {
	bytes := db.Get(blockStoreKey)
	if bytes == nil {
		return BlockStoreStateJSON{
			Height: 0,
		}
	}
	bsj := BlockStoreStateJSON{}
	if err := json.Unmarshal(bytes, &bsj); err != nil {
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

// GetUtxo will search the utxo in db
func (s *Store) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return getUtxo(s.db, hash)
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

// GetTransactionsUtxo will return all the utxo that related to the input txs
func (s *Store) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return getTransactionsUtxo(s.db, view, txs)
}

// GetStoreStatus return the BlockStoreStateJSON
func (s *Store) GetStoreStatus() BlockStoreStateJSON {
	return loadBlockStoreStateJSON(s.db)
}

// GetMainchain read the mainchain map from db
func (s *Store) GetMainchain(hash *bc.Hash) (map[uint64]*bc.Hash, error) {
	return getMainchain(s.db, hash)
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

// SaveChainStatus save the core's newest status && delete old status
func (s *Store) SaveChainStatus(block *legacy.Block, view *state.UtxoViewpoint, m map[uint64]*bc.Hash) error {
	hash := block.Hash()
	batch := s.db.NewBatch()

	if err := saveMainchain(batch, m, &hash); err != nil {
		return err
	}

	if err := saveUtxoView(batch, view); err != nil {
		return err
	}

	BlockStoreStateJSON{Height: block.Height, Hash: &hash}.save(batch)
	batch.Write()

	cleanMainchainDB(s.db, &hash)
	return nil
}
