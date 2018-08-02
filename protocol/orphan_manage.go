package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"sort"
)

var defaultOrphanCap int = 1024

// OrphanManage is use to handle all the orphan block
type OrphanManage struct {
	orphan      map[bc.Hash]*types.Block
	prevOrphans map[bc.Hash][]*bc.Hash
	cap         int
	mtx         sync.RWMutex
}

// NewOrphanManage return a new orphan block
func NewOrphanManage() *OrphanManage {
	return &OrphanManage{
		orphan:      make(map[bc.Hash]*types.Block, defaultOrphanCap),
		prevOrphans: make(map[bc.Hash][]*bc.Hash, defaultOrphanCap),
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
	var lruBlockHash *bc.Hash

	o.mtx.Lock()
	if _, ok := o.orphan[blockHash]; ok {
		return
	}

	o.cap++
	o.orphan[blockHash] = block
	o.prevOrphans[block.PreviousBlockHash] = append(o.prevOrphans[block.PreviousBlockHash], &blockHash)

	// if the cap is full then the old one will be recycled
	if o.cap >= defaultOrphanCap {
		var blocks []*types.Block
		for _, b := range o.orphan {
			blocks = append(blocks, b)
		}
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Height < blocks[j].Height
		})

		oldBlockHash := blocks[0].Hash()
		lruBlockHash = &oldBlockHash
	}
	o.mtx.Unlock()

	if lruBlockHash != nil {
		o.Delete(lruBlockHash)
	}

	log.WithFields(log.Fields{"hash": blockHash.String(), "height": block.Height}).Info("add block to orphan")
}

// Delete will delete the block from OrphanManage
func (o *OrphanManage) Delete(hash *bc.Hash) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	block, ok := o.orphan[*hash]
	if !ok {
		return
	}
	o.cap--
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
