// MemStore is a Store implementation that
// keeps all blockchain state in memory.
//
// It is used in tests to avoid needing a database.
package blockchain

import (
	"fmt"
	"sync"

	"github.com/bytom/protocol/bc/legacy"
	//	"github.com/blockchain/protocol/state"
)

// MemStore satisfies the Store interface.
type MemStore struct {
	mu     sync.Mutex
	Blocks map[uint64]*legacy.Block
	//	State       *state.Snapshot
	//	StateHeight uint64
}

// New returns a new MemStore
func NewMemStore() *MemStore {
	return &MemStore{Blocks: make(map[uint64]*legacy.Block)}
}

func (m *MemStore) Height() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	return uint64(len(m.Blocks))
}

func (m *MemStore) SaveBlock(b *legacy.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.Blocks[b.Height]
	if ok && existing.Hash() != b.Hash() {
		return fmt.Errorf("already have a block at height %d", b.Height)
	}
	m.Blocks[b.Height] = b
	return nil
}

/*
func (m *MemStore) SaveSnapshot(ctx context.Context, height uint64, snapshot *state.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.State = state.Copy(snapshot)
	m.StateHeight = height
	return nil
}
*/

func (m *MemStore) LoadBlock(height uint64) *legacy.Block {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.Blocks[height]
	if !ok {
		return nil
	}
	return b
}

/*
func (m *MemStore) LatestSnapshot(context.Context) (*state.Snapshot, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.State == nil {
		m.State = state.Empty()
	}
	return state.Copy(m.State), m.StateHeight, nil
}
*/

func (m *MemStore) FinalizeBlock(uint64) error { return nil }
