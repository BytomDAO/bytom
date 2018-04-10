package protocol

import (
	"sync"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// OrphanManage is use to handle all the orphan block
type OrphanManage struct {
	//TODO: add orphan cached block limit
	orphan     map[bc.Hash]*types.Block
	preOrphans map[bc.Hash][]*bc.Hash
	mtx        sync.RWMutex
}

// NewOrphanManage return a new orphan block
func NewOrphanManage() *OrphanManage {
	return &OrphanManage{
		orphan:     make(map[bc.Hash]*types.Block),
		preOrphans: make(map[bc.Hash][]*bc.Hash),
	}
}

// BlockExist check is the block in OrphanManage
func (o *OrphanManage) BlockExist(hash *bc.Hash) bool {
	o.mtx.RLock()
	_, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return ok
}

// Add will add the block to OrphanManage
func (o *OrphanManage) Add(block *types.Block) {
	blockHash := block.Hash()
	o.mtx.Lock()
	defer o.mtx.Unlock()

	if _, ok := o.orphan[blockHash]; ok {
		return
	}

	o.orphan[blockHash] = block
	o.preOrphans[block.PreviousBlockHash] = append(o.preOrphans[block.PreviousBlockHash], &blockHash)
}

// Delete will delelte the block from OrphanManage
func (o *OrphanManage) Delete(hash *bc.Hash) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	block, ok := o.orphan[*hash]
	if !ok {
		return
	}
	delete(o.orphan, *hash)

	preOrphans, ok := o.preOrphans[block.PreviousBlockHash]
	if !ok || len(preOrphans) == 1 {
		delete(o.preOrphans, block.PreviousBlockHash)
		return
	}

	for i, preOrphan := range preOrphans {
		if preOrphan == hash {
			o.preOrphans[block.PreviousBlockHash] = append(preOrphans[:i], preOrphans[i+1:]...)
			return
		}
	}
}

// Get return the orphan block by hash
func (o *OrphanManage) Get(hash *bc.Hash) (*types.Block, bool) {
	o.mtx.RLock()
	block, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return block, ok
}
