package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// OrphanManage is use to handle all the orphan block
type OrphanManage struct {
	//TODO: add orphan cached block limit
	orphan      map[bc.Hash]*types.Block
	prevOrphans map[bc.Hash][]*bc.Hash
	mtx         sync.RWMutex
}

// NewOrphanManage return a new orphan block
func NewOrphanManage() *OrphanManage {
	return &OrphanManage{
		orphan:      make(map[bc.Hash]*types.Block),
		prevOrphans: make(map[bc.Hash][]*bc.Hash),
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
	o.prevOrphans[block.PreviousBlockHash] = append(o.prevOrphans[block.PreviousBlockHash], &blockHash)

	log.WithFields(log.Fields{"hash": blockHash.String(), "height": block.Height}).Info("add block to orphan")
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

	prevOrphans, ok := o.prevOrphans[block.PreviousBlockHash]
	if !ok || len(prevOrphans) == 1 {
		delete(o.prevOrphans, block.PreviousBlockHash)
		return
	}

	for i, preOrphan := range prevOrphans {
		if preOrphan == hash {
			o.prevOrphans[block.PreviousBlockHash] = append(prevOrphans[:i], prevOrphans[i+1:]...)
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

// GetPrevOrphans return the list of child orphans
func (o *OrphanManage) GetPrevOrphans(hash *bc.Hash) ([]*bc.Hash, bool) {
	o.mtx.RLock()
	prevOrphans, ok := o.prevOrphans[*hash]
	o.mtx.RUnlock()
	return prevOrphans, ok
}
