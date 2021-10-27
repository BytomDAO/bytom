package contract

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const maxAdvanceTraceBlockNum = 3600

var (
	errGivenTxTooEarly      = errors.New("given tx exceed the max num of blocks ahead")
	errTxAndBlockIsMismatch = errors.New("given tx hash and block hash is mismatch")
	errTxNotIncludeContract = errors.New("input of tx not include utxo contract")
)

type TraceService struct {
	sync.RWMutex
	tracer           *tracer
	infra            *Infrastructure
	scheduler        *traceScheduler
	unconfirmedIndex map[bc.Hash]*TreeNode
	endedInstances   map[string]bool
	bestHeight       uint64
	bestHash         bc.Hash
}

func NewTraceService(infra *Infrastructure) *TraceService {
	allInstances, err := infra.Repository.LoadInstances()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("load instances from db")
	}

	chainStatus := initChainStatus(infra)
	scheduler := newTraceScheduler(infra)
	inSyncInstances := dispatchInstances(allInstances, scheduler, infra.Chain.FinalizedHeight())

	service := &TraceService{
		infra:            infra,
		tracer:           newTracer(inSyncInstances),
		scheduler:        scheduler,
		unconfirmedIndex: make(map[bc.Hash]*TreeNode),
		endedInstances:   make(map[string]bool),
		bestHeight:       chainStatus.BlockHeight,
		bestHash:         chainStatus.BlockHash,
	}
	scheduler.start(service)
	return service
}

func initChainStatus(infra *Infrastructure) *ChainStatus {
	chainStatus := infra.Repository.GetChainStatus()
	if chainStatus == nil {
		bestHeight, bestHash := infra.Chain.BestChain()
		chainStatus = &ChainStatus{BlockHeight: bestHeight, BlockHash: bestHash}
		if err := infra.Repository.SaveChainStatus(chainStatus); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("init chain status for trace service")
		}
	}
	return chainStatus
}

func dispatchInstances(instances []*Instance, scheduler *traceScheduler, finalizedHeight uint64) []*Instance {
	var result []*Instance
	for _, inst := range instances {
		if inst.Status == InSync {
			result = append(result, inst)
		} else if inst.Status == Ended {
			if inst.EndedHeight < finalizedHeight {
				result = append(result, inst)
			}
		} else if inst.Status == Lagging {
			if err := scheduler.addNewJob(inst); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("add new job when init tracer")
			}
		}
	}
	return result
}

func (t *TraceService) BestHeight() uint64 {
	t.RLock()
	defer t.RUnlock()
	return t.bestHeight
}

func (t *TraceService) BestHash() bc.Hash {
	t.RLock()
	defer t.RUnlock()
	return t.bestHash
}

func (t *TraceService) ApplyBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	newInstances := t.tracer.applyBlock(block)
	t.processEndedInstances(newInstances)
	t.bestHeight++
	t.bestHash = block.Hash()
	return t.infra.Repository.SaveInstancesWithStatus(newInstances, t.bestHeight, t.bestHash)
}

func (t *TraceService) DetachBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	newInstances := t.tracer.detachBlock(block)
	t.processEndedInstances(nil)
	t.bestHeight--
	t.bestHash = block.PreviousBlockHash
	return t.infra.Repository.SaveInstancesWithStatus(newInstances, t.bestHeight, t.bestHash)
}

func (t *TraceService) AddUnconfirmedTx(tx *types.Tx) {
	transfers := parseTransfers(tx)
	for _, transfer := range transfers {
		inUTXOs, outUTXOs := transfer.inUTXOs, transfer.outUTXOs
		if len(inUTXOs) == 0 || len(outUTXOs) == 0 {
			return
		}

		treeNode := &TreeNode{TxHash: tx.ID, UTXOs: outUTXOs}
		if inst := t.tracer.index.getByUTXO(inUTXOs[0].Hash); inst != nil {
			inst.Unconfirmed = append(inst.Unconfirmed, treeNode)
			t.addToUnconfirmedIndex(treeNode, outUTXOs)
			return
		}

		if parent, ok := t.unconfirmedIndex[inUTXOs[0].Hash]; ok {
			parent.Children = append(parent.Children, treeNode)
			t.addToUnconfirmedIndex(treeNode, outUTXOs)
		}
	}
}

func (t *TraceService) CreateInstance(txHash, blockHash bc.Hash) ([]string, error) {
	block, err := t.infra.Chain.GetBlockByHash(&blockHash)
	if err != nil {
		return nil, err
	}

	if bestHeight, _ := t.infra.Chain.BestChain(); bestHeight-block.Height > maxAdvanceTraceBlockNum {
		return nil, errGivenTxTooEarly
	}

	tx := findTx(block, txHash)
	if tx == nil {
		return nil, errTxAndBlockIsMismatch
	}

	transfers := parseTransfers(tx)
	if len(transfers) == 0 {
		return nil, errTxNotIncludeContract
	}

	var traceIDs []string
	for _, transfer := range transfers {
		inst := newInstance(transfer, block)
		traceIDs = append(traceIDs, inst.TraceID)
		if err := t.addNewTraceJob(inst); err != nil {
			return nil, err
		}
	}
	return traceIDs, nil
}

func (t *TraceService) RemoveInstance(traceID string) error {
	t.Lock()
	defer t.Unlock()

	t.infra.Repository.RemoveInstance(traceID)
	t.tracer.removeInstance(traceID)
	return nil
}

func (t *TraceService) GetInstance(traceID string) (*Instance, error) {
	return t.infra.Repository.GetInstance(traceID)
}

func (t *TraceService) takeOverInstances(instances []*Instance, blockHash bc.Hash) bool {
	t.Lock()
	defer t.Unlock()

	if blockHash != t.bestHash {
		return false
	}

	for _, inst := range instances {
		if inst.Status != Ended {
			inst.Status = InSync
		}
	}

	if err := t.infra.Repository.SaveInstances(instances); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("save instances when take over instances")
		return false
	}

	t.tracer.addInstances(instances)
	t.processEndedInstances(instances)
	return true
}

func (t *TraceService) processEndedInstances(instances []*Instance) {
	for _, inst := range instances {
		if inst.Status == Ended {
			t.endedInstances[inst.TraceID] = true
		}
	}

	finalizedHeight := t.infra.Chain.FinalizedHeight()
	for traceID := range t.endedInstances {
		inst := t.tracer.getInstance(traceID)
		if inst.Status != Ended {
			delete(t.endedInstances, traceID)
		} else if finalizedHeight >= inst.EndedHeight {
			delete(t.endedInstances, traceID)
			t.tracer.removeInstance(traceID)
		}
	}
}

func (t *TraceService) addNewTraceJob(inst *Instance) error {
	if err := t.infra.Repository.SaveInstances([]*Instance{inst}); err != nil {
		return err
	}

	if inst.Status != Ended {
		if err := t.scheduler.addNewJob(inst); err != nil {
			return err
		}
	}
	return nil
}

func (t *TraceService) addToUnconfirmedIndex(treeNode *TreeNode, utxos []*UTXO) {
	for _, utxo := range utxos {
		t.unconfirmedIndex[utxo.Hash] = treeNode
	}
}

func findTx(block *types.Block, txHash bc.Hash) *types.Tx {
	for _, tx := range block.Transactions {
		if tx.ID == txHash {
			return tx
		}
	}
	return nil
}
