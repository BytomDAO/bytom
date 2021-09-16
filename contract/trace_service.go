package contract

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"

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
	unconfirmedIndex map[bc.Hash]*treeNode
	bestHeight       uint64
	bestHash         bc.Hash
}

func NewTraceService(infra *Infrastructure) *TraceService {
	allInstances, err := infra.Repository.LoadInstances()
	if err != nil {
		logrus.WithField("err", err).Fatal("load instances from db")
	}

	chainStatus := infra.Repository.GetChainStatus()
	if chainStatus == nil {
		chainStatus.BlockHeight, chainStatus.BlockHash = infra.Chain.BestChain()
		if err := infra.Repository.SaveChainStatus(chainStatus);  err != nil {
			logrus.WithField("err", err).Fatal("init chain status for trace service")
		}
	}

	scheduler := newTraceScheduler(infra)
	inSyncInstances := dispatchInstances(allInstances, scheduler)

	service := &TraceService{
		infra:            infra,
		tracer:           newTracer(inSyncInstances),
		scheduler:        scheduler,
		unconfirmedIndex: make(map[bc.Hash]*treeNode),
		bestHeight:       chainStatus.BlockHeight,
		bestHash:         chainStatus.BlockHash,
	}
	scheduler.start(service)
	return service
}

func dispatchInstances(instances []*Instance, scheduler *traceScheduler) []*Instance {
	var result []*Instance
	for _, inst := range instances {
		if inst.Status == InSync {
			result = append(result, inst)
		} else if inst.Status == Lagging {
			if err := scheduler.addNewJob(inst); err != nil {
				logrus.WithField("err", err).Fatal("add new job when init tracer")
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
	t.bestHeight++
	t.bestHash = block.Hash()
	return t.infra.Repository.SaveInstancesWithStatus(newInstances, t.bestHeight, t.bestHash)
}

func (t *TraceService) DetachBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	newInstances := t.tracer.detachBlock(block)
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

		treeNode := &treeNode{TxHash: tx.ID, UTXOs: outUTXOs}
		if inst := t.tracer.table.getByUTXO(inUTXOs[0].hash); inst != nil {
			inst.Unconfirmed = append(inst.Unconfirmed, treeNode)
			t.addToUnconfirmedIndex(treeNode, outUTXOs)
			return
		}

		if parent, ok := t.unconfirmedIndex[inUTXOs[0].hash]; ok {
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
		inst := newInstance(transfer.inUTXOs, transfer.outUTXOs)
		traceIDs = append(traceIDs, inst.TraceID)
		if err := t.addNewTraceJob(inst, block); err != nil {
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
	t.RLock()
	defer t.RUnlock()

	if blockHash != t.bestHash {
		return false
	}

	if err := t.infra.Repository.SaveInstances(instances); err != nil {
		logrus.WithField("err", err).Error("save instances when take over instances")
		return false
	}

	t.tracer.addInstances(instances)
	return true
}

func (t *TraceService) addNewTraceJob(inst *Instance, block *types.Block) error {
	if err := t.infra.Repository.SaveInstances([]*Instance{inst}); err != nil {
		return err
	}

	if inst.Status != Finalized {
		inst.ScannedHash = block.Hash()
		inst.ScannedHeight = block.Height
		if err := t.scheduler.addNewJob(inst); err != nil {
			return err
		}
	}
	return nil
}

func (t *TraceService) addToUnconfirmedIndex(treeNode *treeNode, utxos []*UTXO) {
	for _, utxo := range utxos {
		t.unconfirmedIndex[utxo.hash] = treeNode
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
