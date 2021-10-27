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
	currentHeight uint64
	currentHash   bc.Hash
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
		if beginHeight > t.tracerService.BestHeight() {
			continue
		}
		t.tracer = newTracer(jobs[beginHash])

		for t.currentHeight, t.currentHash = beginHeight, beginHash;; {
			if t.currentHeight == t.tracerService.BestHeight() {
				if err := t.finishJobs(jobs); err != nil {
					log.WithFields(log.Fields{"module": logModule, "err": err}).Error("finish jobs")
				} else {
					break
				}
			}

			if ok, err := t.tryAttach(jobs); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("try attach on trace scheduler")
			} else if !ok {
				if err := t.detach(jobs); err != nil {
					log.WithFields(log.Fields{"module": logModule, "err": err}).Error("detach on trace scheduler")
				}
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

func (t *traceScheduler) tryAttach(jobs map[bc.Hash][]*Instance) (bool, error) {
	if t.currentHash == t.tracerService.BestHash() {
		return true, nil
	}

	block, err := t.infra.Chain.GetBlockByHeight(t.currentHeight+1)
	if err != nil {
		return false, err
	}

	if block.PreviousBlockHash != t.currentHash {
		return false, nil
	}

	t.tracer.applyBlock(block)
	t.currentHeight++
	t.currentHash = block.Hash()

	if instances, ok := jobs[block.Hash()]; ok {
		t.tracer.addInstances(instances)
	}
	return true, nil
}

func (t *traceScheduler) detach(jobs map[bc.Hash][]*Instance) error {
	block, err := t.infra.Chain.GetBlockByHash(&t.currentHash)
	if err != nil {
		return err
	}

	if instances, ok := jobs[block.Hash()]; ok {
		for _, inst := range instances {
			t.tracer.removeInstance(inst.TraceID)
		}
	}

	t.tracer.detachBlock(block)
	t.currentHeight--
	t.currentHash = block.PreviousBlockHash
	return nil
}

func (t *traceScheduler) finishJobs(jobs map[bc.Hash][]*Instance) error {
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

	if len(inSyncInstances) != 0 {
		if ok := t.tracerService.takeOverInstances(inSyncInstances, t.currentHash); ok {
			t.releaseInstances(inSyncInstances)
		}
	}
	return nil
}

func (t *traceScheduler) releaseInstances(instances []*Instance) {
	t.weighted.Release(int64(len(instances)))
	for _, inst := range instances {
		t.instances.Delete(inst.TraceID)
	}
}
