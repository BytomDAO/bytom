package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"time"
)

var (
	orphanTTL                = 10 * time.Minute
	orphanExpireScanInterval = 3 * time.Minute
)

type OrphanTx struct {
	orphanBlock *types.Block
	expiration  time.Time
}

// OrphanManage is use to handle all the orphan block
type OrphanManage struct {
	orphan      map[bc.Hash]*OrphanTx
	prevOrphans map[bc.Hash][]*bc.Hash
	mtx         sync.RWMutex
}

// NewOrphanManage return a new orphan block
func NewOrphanManage() *OrphanManage {
	o := &OrphanManage{
		orphan:      make(map[bc.Hash]*OrphanTx),
		prevOrphans: make(map[bc.Hash][]*bc.Hash),
	}

	go o.orphanExpireWorker()
	return o
}

func (o *OrphanManage) orphanExpireWorker() {
	ticker := time.NewTicker(orphanExpireScanInterval)
	for now := range ticker.C {
		o.orphanExpire(now)
	}
	ticker.Stop()
}

func (o *OrphanManage) orphanExpire(now time.Time) {
	var orphans []bc.Hash

	o.mtx.RLock()
	for hash, orphan := range o.orphan {
		if orphan.expiration.Before(now) {
			orphans = append(orphans, hash)
		}
	}
	o.mtx.RUnlock()

	for _, hash := range orphans {
		o.Delete(&hash)
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

	o.orphan[blockHash] = &OrphanTx{block, time.Now().Add(orphanTTL)}
	o.prevOrphans[block.PreviousBlockHash] = append(o.prevOrphans[block.PreviousBlockHash], &blockHash)

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
	delete(o.orphan, *hash)

	prevOrphans, ok := o.prevOrphans[block.orphanBlock.PreviousBlockHash]
	if !ok || len(prevOrphans) == 1 {
		delete(o.prevOrphans, block.orphanBlock.PreviousBlockHash)
		return
	}

	for i, preOrphan := range prevOrphans {
		if preOrphan == hash {
			o.prevOrphans[block.orphanBlock.PreviousBlockHash] = append(prevOrphans[:i], prevOrphans[i+1:]...)
			return
		}
	}
}

// Get return the orphan block by hash
func (o *OrphanManage) Get(hash *bc.Hash) (*types.Block, bool) {
	o.mtx.RLock()
	block, ok := o.orphan[*hash]
	o.mtx.RUnlock()
	return block.orphanBlock, ok
}

// GetPrevOrphans return the list of child orphans
func (o *OrphanManage) GetPrevOrphans(hash *bc.Hash) ([]*bc.Hash, bool) {
	o.mtx.RLock()
	prevOrphans, ok := o.prevOrphans[*hash]
	o.mtx.RUnlock()
	return prevOrphans, ok
}
