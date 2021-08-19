package chainmgr

import (
	"encoding/binary"
	"sync"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc/types"
)

var (
	maxByteOfStorageRAM = 800 * 1024 * 1024 //100MB
	errStorageFindBlock = errors.New("can't find block from storage")
	errDBFindBlock      = errors.New("can't find block from DB")
)

// LocalStore is the interface for persistent storage
type LocalStore interface {
	writeBlock(block *types.Block) error
	readBlock(height uint64) (*types.Block, error)
	clearData()
}

type blockStorage struct {
	block  *types.Block
	peerID string
	size   int
	isRAM  bool
}

type storage struct {
	actualUsage int
	blocks      map[uint64]*blockStorage
	localStore  LocalStore
	mux         sync.RWMutex
}

func newStorage(db dbm.DB) *storage {
	DBStorage := newDBStore(db)
	DBStorage.clearData()
	return &storage{
		blocks:     make(map[uint64]*blockStorage),
		localStore: DBStorage,
	}
}

func (s *storage) writeBlocks(peerID string, blocks []*types.Block) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, block := range blocks {
		binaryBlock, err := block.MarshalText()
		if err != nil {
			return errors.Wrap(err, "Marshal block header")
		}

		if len(binaryBlock)+s.actualUsage < maxByteOfStorageRAM {
			s.blocks[block.Height] = &blockStorage{block: block, peerID: peerID, size: len(binaryBlock), isRAM: true}
			s.actualUsage += len(binaryBlock)
			continue
		}

		if err := s.localStore.writeBlock(block); err != nil {
			return err
		}

		s.blocks[block.Height] = &blockStorage{peerID: peerID, isRAM: false}
	}

	return nil
}

func (s *storage) readBlock(height uint64) (*blockStorage, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	blockStore, ok := s.blocks[height]
	if !ok {
		return nil, errStorageFindBlock
	}

	if blockStore.isRAM {
		return blockStore, nil
	}

	block, err := s.localStore.readBlock(height)
	if err != nil {
		return nil, err
	}

	blockStore.block = block
	return blockStore, nil
}

// deleteBlock delete blocks in memory
func (s *storage) deleteBlock(height uint64) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	blockStore, ok := s.blocks[height]
	if !ok {
		return
	}

	if blockStore.isRAM {
		s.actualUsage -= blockStore.size
		delete(s.blocks, height)
	}
}

func (s *storage) resetParameter() {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.blocks = make(map[uint64]*blockStorage)
	s.actualUsage = 0
	s.localStore.clearData()
}

type levelDBStorage struct {
	db dbm.DB
}

func newDBStore(db dbm.DB) *levelDBStorage {
	return &levelDBStorage{
		db: db,
	}
}

func (ls *levelDBStorage) clearData() {
	iter := ls.db.Iterator()
	defer iter.Release()

	for iter.Next() {
		ls.db.Delete(iter.Key())
	}
}

func (ls *levelDBStorage) writeBlock(block *types.Block) error {
	binaryBlock, err := block.MarshalText()
	if err != nil {
		return err
	}

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, block.Height)
	ls.db.Set(key, binaryBlock)
	return nil
}

func (ls *levelDBStorage) readBlock(height uint64) (*types.Block, error) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	binaryBlock := ls.db.Get(key)
	if binaryBlock == nil {
		return nil, errDBFindBlock
	}

	block := &types.Block{}
	if err := block.UnmarshalText(binaryBlock); err != nil {
		return nil, err
	}

	return block, nil
}
