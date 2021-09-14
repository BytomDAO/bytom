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

type TracerService struct {
	sync.RWMutex
	tracer           *tracer
	infra            *Infrastructure
	scheduler        *traceScheduler
	unconfirmedIndex map[bc.Hash]*treeNode
}

func NewTracerService(infra *Infrastructure) *TracerService {
	allInstances, err := infra.Repository.LoadInstances()
	if err != nil {
		logrus.WithField("err", err).Fatal("load instances from db")
	}

	scheduler := newTraceScheduler(infra)

	var instances []*Instance
	for _, inst := range allInstances {
		if inst.Status == InSync {
			instances = append(instances, inst)
		} else if inst.Status == Lagging {
			if err := scheduler.addNewJob(inst); err != nil {
				logrus.WithField("err", err).Fatal("add new job when init tracer")
			}
		}
	}
	service := &TracerService{
		infra:            infra,
		tracer:           newTracer(instances),
		scheduler:        scheduler,
		unconfirmedIndex: make(map[bc.Hash]*treeNode),
	}
	scheduler.start(service)
	return service
}

func (t *TracerService) ApplyBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	newInstances := t.tracer.applyBlock(block)
	return t.infra.Repository.SaveInstances(newInstances)
}

func (t *TracerService) DetachBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	newInstances := t.tracer.detachBlock(block)
	return t.infra.Repository.SaveInstances(newInstances)
}

func (t *TracerService) AddUnconfirmedTx(tx *types.Tx) {
	transfers := t.tracer.parseTransfers(tx)
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

func (t *TracerService) CreateInstance(txHash, blockHash bc.Hash) ([]string, error) {
	block, err := t.infra.Chain.GetBlockByHash(&blockHash)
	if err != nil {
		return nil, err
	}

	if bestHeight := t.infra.Chain.BestBlockHeight(); bestHeight-block.Height > maxAdvanceTraceBlockNum {
		return nil, errGivenTxTooEarly
	}

	tx := findTx(block, txHash)
	if tx == nil {
		return nil, errTxAndBlockIsMismatch
	}

	transfers := t.tracer.parseTransfers(tx)
	if len(transfers) == 0 {
		return nil, errTxNotIncludeContract
	}

	var traceIDs []string
	for _, transfer := range transfers {
		inst := NewInstance(transfer.inUTXOs, transfer.outUTXOs)
		traceIDs = append(traceIDs, inst.TraceID)
		if err := t.addNewTraceJob(inst, block); err != nil {
			return nil, err
		}
	}
	return traceIDs, nil
}

func (t *TracerService) RemoveInstance(traceID string) error {
	t.Lock()
	defer t.Unlock()

	t.infra.Repository.RemoveInstance(traceID)
	t.tracer.removeInstance(traceID)
	return nil
}

func (t *TracerService) GetInstance(traceID string) (*Instance, error) {
	return t.infra.Repository.GetInstance(traceID)
}

func (t *TracerService) takeOverInstances(instances []*Instance, blockHash bc.Hash) bool {
	t.RLock()
	defer t.RUnlock()

	if bestHash := t.infra.Chain.BestBlockHash(); blockHash != *bestHash {
		return false
	}

	if err := t.infra.Repository.SaveInstances(instances); err != nil {
		logrus.WithField("err", err).Error("save instances when take over instances")
		return false
	}

	t.tracer.addInstances(instances)
	return true
}

func (t *TracerService) addNewTraceJob(inst *Instance, block *types.Block) error {
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

func (t *TracerService) addToUnconfirmedIndex(treeNode *treeNode, utxos []*UTXO) {
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
