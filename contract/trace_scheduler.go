package contract

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/bytom/bytom/protocol/bc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

var errInstQueueOverflow = errors.New("instance queue is overflow")

type traceScheduler struct {
	weighted      *semaphore.Weighted
	instances     *sync.Map
	tracerService *TraceService
	infra         *Infrastructure
	tracer        *tracer
}

func newTraceScheduler(infra *Infrastructure) *traceScheduler {
	scheduler := &traceScheduler{
		weighted:  semaphore.NewWeighted(1000),
		instances: new(sync.Map),
		infra:     infra,
	}
	return scheduler
}

func (t *traceScheduler) start(service *TraceService) {
	t.tracerService = service
	go t.processLoop()
}

func (t *traceScheduler) addNewJob(instance *Instance) error {
	if !t.weighted.TryAcquire(1) {
		return errInstQueueOverflow
	}

	t.instances.Store(instance.TraceID, instance)
	return nil
}

func (t *traceScheduler) processLoop() {
	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		jobs, beginHeight := t.prepareJobs()
		if len(jobs) == 0 {
			continue
		}

		t.tracer = newTracer(nil)

		var prevHash *bc.Hash
		catchedJobs := make(map[bc.Hash][]*Instance)
		for height := beginHeight + 1; ; height++ {
			if ok, err := t.tryAttach(height, prevHash, jobs, catchedJobs); err != nil {
				log.WithField("err", err).Error("try attach on trace scheduler")
				break
			} else if !ok {
				if err := t.detach(prevHash, catchedJobs); err != nil {
					log.WithField("err", err).Error("detach on trace scheduler")
					break
				}
				height -= 2
			}
			if bestHeight := t.tracerService.BestHeight(); height == bestHeight {
				if err := t.finishJobs(jobs, catchedJobs, *prevHash); err != nil {
					log.WithField("err", err).Error("finish jobs")
					break
				}
			}
		}
	}
}

func (t *traceScheduler) prepareJobs() (map[bc.Hash][]*Instance, uint64) {
	var beginHeight uint64 = math.MaxUint64
	hashToJobs := make(map[bc.Hash][]*Instance)
	t.instances.Range(func(_, value interface{}) bool {
		inst := value.(*Instance)
		hashToJobs[inst.ScannedHash] = append(hashToJobs[inst.ScannedHash], inst)
		if inst.ScannedHeight < beginHeight {
			beginHeight = inst.ScannedHeight
		}
		return true
	})
	return hashToJobs, beginHeight
}

func (t *traceScheduler) tryAttach(height uint64, prevHash *bc.Hash, jobs, catchedJobs map[bc.Hash][]*Instance) (bool, error) {
	block, err := t.infra.Chain.GetBlockByHeight(height)
	if err != nil {
		return false, err
	}

	if prevHash != nil && block.PreviousBlockHash != *prevHash {
		return false, nil
	}

	if instances, ok := jobs[block.PreviousBlockHash]; ok {
		t.tracer.addInstances(instances)
		catchedJobs[block.PreviousBlockHash] = instances
	}

	t.tracer.applyBlock(block)
	*prevHash = block.Hash()
	return true, nil
}

func (t *traceScheduler) detach(prevHash *bc.Hash, catchedJobs map[bc.Hash][]*Instance) error {
	prevBlock, err := t.infra.Chain.GetBlockByHash(prevHash)
	if err != nil {
		return err
	}

	if instances, ok := catchedJobs[prevBlock.Hash()]; ok {
		for _, inst := range instances {
			t.tracer.removeInstance(inst.TraceID)
		}
		delete(catchedJobs, prevBlock.Hash())
	}

	t.tracer.detachBlock(prevBlock)
	*prevHash = prevBlock.PreviousBlockHash
	return nil
}

func (t *traceScheduler) finishJobs(jobs, catchedJobs map[bc.Hash][]*Instance, scannedHash bc.Hash) error {
	var inSyncInstances, offChainInstances []*Instance
	for hash, instances := range jobs {
		if _, ok := catchedJobs[hash]; !ok {
			offChainInstances = append(offChainInstances, instances...)
			for _, inst := range instances {
				inst.Status = OffChain
			}
		} else {
			inSyncInstances = append(inSyncInstances, instances...)
		}
	}

	if err := t.infra.Repository.SaveInstances(offChainInstances); err != nil {
		return err
	}

	t.releaseInstances(offChainInstances)

	if ok := t.tracerService.takeOverInstances(inSyncInstances, scannedHash); ok {
		t.releaseInstances(inSyncInstances)
	}
	return nil
}

func (t *traceScheduler) releaseInstances(instances []*Instance) {
	t.weighted.Release(int64(len(instances)))
	for _, inst := range instances {
		t.instances.Delete(inst.TraceID)
	}
}
