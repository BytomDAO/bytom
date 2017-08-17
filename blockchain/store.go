package blockchain

import (
	"encoding/json"
	"fmt"
	"sync"

	. "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
    "github.com/bytom/protocol/bc/legacy"
)

type BlockStore struct {
	db dbm.DB

	mtx    sync.RWMutex
	height uint64
}

func NewBlockStore(db dbm.DB) *BlockStore {
	bsjson := LoadBlockStoreStateJSON(db)
	return &BlockStore{
		height: bsjson.Height,
		db:     db,
	}
}

// Height() returns the last known contiguous block height.
func (bs *BlockStore) Height() uint64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.height
}

func (bs *BlockStore) LoadBlock(height uint64) *legacy.Block {
    var block *legacy.Block
    bytez := bs.db.Get(calcBlockKey(height))
    if bytez == nil {
        return nil
    }
    block.UnmarshalText(bytez)
	return block
}

func (bs *BlockStore) SaveBlock(block *legacy.Block) {
	height := block.Height
//	if height != bs.Height()+1 {
//		PanicSanity(Fmt("BlockStore can only save contiguous blocks. Wanted %v, got %v", bs.Height()+1, height))
//	}

    binaryBlock, err := block.MarshalText()
    if err != nil {
        PanicCrisis(Fmt("Error Marshal block meta: %v", err))
    }
    bs.db.Set(calcBlockKey(height), binaryBlock)

	// Save new BlockStoreStateJSON descriptor
	BlockStoreStateJSON{Height: height}.Save(bs.db)

	// Done!
	bs.mtx.Lock()
	bs.height = height
	bs.mtx.Unlock()

	// Flush
	bs.db.SetSync(nil, nil)
}

//-----------------------------------------------------------------------------

func calcBlockHeadKey(height uint64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func calcBlockKey(height uint64) []byte {
	return []byte(fmt.Sprintf("B:%v", height))
}

//-----------------------------------------------------------------------------

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
