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
		jobs, beginHeight, beginHash := t.prepareJobs()
		if len(jobs) == 0 {
			continue
		}

		t.tracer = newTracer(jobs[beginHash])

		for height, blockHash := beginHeight, beginHash; ; height++ {
			if bestHeight := t.tracerService.BestHeight(); height == bestHeight {
				if err := t.finishJobs(jobs, blockHash); err != nil {
					log.WithField("err", err).Error("finish jobs")
					break
				}
			}

			if ok, err := t.tryAttach(height+1, &blockHash, jobs); err != nil {
				log.WithField("err", err).Error("try attach on trace scheduler")
				break
			} else if !ok {
				if err := t.detach(&blockHash, jobs); err != nil {
					log.WithField("err", err).Error("detach on trace scheduler")
					break
				}
				height -= 2
			}
		}
	}
}

func (t *traceScheduler) prepareJobs() (map[bc.Hash][]*Instance, uint64, bc.Hash) {
	beginHeight, beginHash := uint64(math.MaxUint64), bc.Hash{}
	hashToJobs := make(map[bc.Hash][]*Instance)
	t.instances.Range(func(_, value interface{}) bool {
		inst := value.(*Instance)
		hashToJobs[inst.ScannedHash] = append(hashToJobs[inst.ScannedHash], inst)
		if inst.ScannedHeight < beginHeight {
			beginHeight = inst.ScannedHeight
			beginHash = inst.ScannedHash
		}
		return true
	})
	return hashToJobs, beginHeight, beginHash
}

func (t *traceScheduler) tryAttach(height uint64, blockHash *bc.Hash, jobs map[bc.Hash][]*Instance) (bool, error) {
	block, err := t.infra.Chain.GetBlockByHeight(height)
	if err != nil {
		return false, err
	}

	if block.PreviousBlockHash != *blockHash {
		return false, nil
	}

	t.tracer.applyBlock(block)
	*blockHash = block.Hash()

	if instances, ok := jobs[block.Hash()]; ok {
		t.tracer.addInstances(instances)
	}
	return true, nil
}

func (t *traceScheduler) detach(blockHash *bc.Hash, jobs map[bc.Hash][]*Instance) error {
	block, err := t.infra.Chain.GetBlockByHash(blockHash)
	if err != nil {
		return err
	}

	if instances, ok := jobs[block.Hash()]; ok {
		for _, inst := range instances {
			t.tracer.removeInstance(inst.TraceID)
		}
	}

	t.tracer.detachBlock(block)
	*blockHash = block.PreviousBlockHash
	return nil
}

func (t *traceScheduler) finishJobs(jobs map[bc.Hash][]*Instance, scannedHash bc.Hash) error {
	inSyncInstances := t.tracer.allInstances()
	inSyncMap := make(map[string]bool)
	for _, inst := range inSyncInstances {
		inSyncMap[inst.TraceID] = true
	}

	var offChainInstances []*Instance
	for _, instances := range jobs {
		for _, inst := range instances {
			if _, ok := inSyncMap[inst.TraceID]; !ok {
				inst.Status = OffChain
				offChainInstances = append(offChainInstances, inst)
			}
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
