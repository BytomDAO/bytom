package protocol

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

var (
	orphanBlockTTL           = 60 * time.Minute
	orphanExpireWorkInterval = 3 * time.Minute
)

type orphanBlock struct {
	*types.Block
	expiration time.Time
}

// OrphanManage is use to handle all the orphan block
type OrphanManage struct {
	orphan      map[bc.Hash]*orphanBlock
	prevOrphans map[bc.Hash][]*bc.Hash
	mtx         sync.RWMutex
}

// NewOrphanManage return a new orphan block
func NewOrphanManage() *OrphanManage {
	o := &OrphanManage{
		orphan:      make(map[bc.Hash]*orphanBlock),
		prevOrphans: make(map[bc.Hash][]*bc.Hash),
	}

	go o.orphanExpireWorker()
	return o
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

	o.orphan[blockHash] = &orphanBlock{block, time.Now().Add(orphanBlockTTL)}
	o.prevOrphans[block.PreviousBlockHash] = append(o.prevOrphans[block.PreviousBlockHash], &blockHash)

	log.WithFields(log.Fields{"hash": blockHash.String(), "height": block.Height}).Info("add block to orphan")
}

// Delete will delete the block from OrphanManage
func (o *OrphanManage) Delete(hash *bc.Hash) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	o.delete(hash)
}

// Get return the orphan block by hash
func (o *OrphanManage) Get(hash *bc.Hash) (*types.Block, bool) {
	o.mtx.RLock()
	block, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return block.Block, ok
}

// GetPrevOrphans return the list of child orphans
func (o *OrphanManage) GetPrevOrphans(hash *bc.Hash) ([]*bc.Hash, bool) {
	o.mtx.RLock()
	prevOrphans, ok := o.prevOrphans[*hash]
	o.mtx.RUnlock()
	return prevOrphans, ok
}

func (o *OrphanManage) delete(hash *bc.Hash) {
	block, ok := o.orphan[*hash]
	if !ok {
		return
	}
	delete(o.orphan, *hash)

	prevOrphans, ok := o.prevOrphans[block.Block.PreviousBlockHash]
	if !ok || len(prevOrphans) == 1 {
		delete(o.prevOrphans, block.Block.PreviousBlockHash)
		return
	}

	for i, preOrphan := range prevOrphans {
		if preOrphan == hash {
			o.prevOrphans[block.Block.PreviousBlockHash] = append(prevOrphans[:i], prevOrphans[i+1:]...)
			return
		}
	}
}

func (o *OrphanManage) orphanExpireWorker() {
	ticker := time.NewTicker(orphanExpireWorkInterval)
	for now := range ticker.C {
		o.orphanExpire(now)
	}
	ticker.Stop()
}

func (o *OrphanManage) orphanExpire(now time.Time) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	for hash, orphan := range o.orphan {
		if orphan.expiration.Before(now) {
			o.delete(&hash)
		}
	}
}
