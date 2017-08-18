package txdb

import (
	"context"
	"fmt"
	"encoding/json"

	"github.com/bytom/errors"
	//"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
    dbm "github.com/tendermint/tmlibs/db"
	. "github.com/tendermint/tmlibs/common"

)

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type Store struct {
	db dbm.DB

	cache blockCache
}


//var _ protocol.Store = (*Store)(nil)

func calcBlockHeadKey(height uint64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func calcBlockKey(height uint64) []byte {
	return []byte(fmt.Sprintf("B:%v", height))
}

func LoadBlock(db dbm.DB, height uint64) *legacy.Block {
    var block *legacy.Block
    bytez := db.Get(calcBlockKey(height))
    if bytez == nil {
        return nil
    }
    block.UnmarshalText(bytez)
	return block
}

var blockStoreKey = []byte("blockStore")

type BlockStoreStateJSON struct {
	Height uint64
}

func (bsj BlockStoreStateJSON) Save(db dbm.DB) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		PanicSanity(Fmt("Could not marshal state bytes: %v", err))
	}
	db.SetSync(blockStoreKey, bytes)
}

func LoadBlockStoreStateJSON(db dbm.DB) BlockStoreStateJSON {
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


// NewStore creates and returns a new Store object.
//
// For testing purposes, it is usually much faster
// and more convenient to use package bytom/protocol/memstore
// instead.
func NewStore(db dbm.DB) *Store {
	cache := newBlockCache(func(height uint64) *legacy.Block {
			return LoadBlock(db, height)
		})
	return &Store{
		db: db,
		cache: cache,
		}
}

// Height returns the height of the blockchain.
func (s *Store) Height() uint64 {
	heightJson := LoadBlockStoreStateJSON(s.db)
    return heightJson.Height
}

// GetBlock looks up the block with the provided block height.
// If no block is found at that height, it returns an error.

func (s *Store) GetBlock(height uint64) (*legacy.Block, error) {
	return s.cache.lookup(height)
}

// LatestSnapshot returns the most recent state snapshot stored in
// the database and its corresponding block height.
func (s *Store) LatestSnapshot(ctx context.Context) (*state.Snapshot, uint64, error) {
	return getStateSnapshot(ctx, s.db)
}

/*
// LatestSnapshotInfo returns the height and size of the most recent
// state snapshot stored in the database.
func (s *Store) LatestSnapshotInfo(ctx context.Context) (height uint64, size uint64, err error) {
	const q = `
		SELECT height, octet_length(data) FROM snapshots ORDER BY height DESC LIMIT 1
	`
	err = s.db.QueryRowContext(ctx, q).Scan(&height, &size)
	return height, size, err
}

// GetSnapshot returns the state snapshot stored at the provided height,
// in Chain Core's binary protobuf representation. If no snapshot exists
// at the provided height, an error is returned.
func (s *Store) GetSnapshot(ctx context.Context, height uint64) ([]byte, error) {
	return getRawSnapshot(ctx, s.db, height)
}
*/

// SaveBlock persists a new block in the database.
func (s *Store) SaveBlock(block *legacy.Block) error {
	s.cache.add(block)
	height := block.Height

    binaryBlock, err := block.MarshalText()
    if err != nil {
        PanicCrisis(Fmt("Error Marshal block meta: %v", err))
    }
    s.db.Set(calcBlockKey(height), binaryBlock)

	// Save new BlockStoreStateJSON descriptor
	BlockStoreStateJSON{Height: height}.Save(s.db)

	// Flush
	s.db.SetSync(nil, nil)

	return nil
}

// SaveSnapshot saves a state snapshot to the database.
func (s *Store) SaveSnapshot(ctx context.Context, height uint64, snapshot *state.Snapshot) error {
	err := storeStateSnapshot(ctx, s.db, snapshot, height)
	return errors.Wrap(err, "saving state tree")
}

func (s *Store) FinalizeBlock(ctx context.Context, height uint64) error {
//	_, err := s.db.ExecContext(ctx, `SELECT pg_notify('newblock', $1)`, height)
	return nil
}
