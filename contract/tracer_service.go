package contract

import (
	"errors"
	"sync"

	"github.com/google/uuid"
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
	unconfirmedIndex map[bc.Hash]*TreeNode
}

func NewTracerService(infra *Infrastructure, scheduler *traceScheduler) *TracerService {
	allInstances, err := infra.Repository.LoadInstances()
	if err != nil {
		logrus.WithField("err", err).Fatal("load instances from db")
	}

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
	return &TracerService{
		infra:            infra,
		tracer:           newTracer(instances),
		scheduler:        scheduler,
		unconfirmedIndex: make(map[bc.Hash]*TreeNode),
	}
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
	inUTXOs, outUTXOs := parseTransfer(tx)
	if len(inUTXOs) == 0 || len(outUTXOs) == 0 {
		return
	}

	treeNode := &TreeNode{TxHash: tx.ID, UTXOs: outUTXOs}
	if inst := t.tracer.table.GetByUTXO(inUTXOs[0].hash); inst != nil {
		inst.Unconfirmed = append(inst.Unconfirmed, treeNode)
		t.addToUnconfirmedIndex(treeNode, outUTXOs)
		return
	}

	if parent, ok := t.unconfirmedIndex[inUTXOs[0].hash]; ok {
		parent.Children = append(parent.Children, treeNode)
		t.addToUnconfirmedIndex(treeNode, outUTXOs)
	}
}

func (t *TracerService) CreateInstance(txHash, blockHash bc.Hash) (string, error) {
	block, err := t.infra.Chain.GetBlockByHash(&blockHash)
	if err != nil {
		return "", err
	}

	if bestHeight := t.infra.Chain.BestBlockHeight(); bestHeight-block.Height > maxAdvanceTraceBlockNum {
		return "", errGivenTxTooEarly
	}

	tx := findTx(block, txHash)
	if tx == nil {
		return "", errTxAndBlockIsMismatch
	}

	inUTXOs, outUTXOs := parseTransfer(tx)
	if len(inUTXOs) == 0 && len(outUTXOs) == 0 {
		return "", errTxNotIncludeContract
	}

	inst := NewInstance(uuid.New().String(), inUTXOs, outUTXOs)
	if err := t.addNewTraceJob(inst, block); err != nil {
		return "", err
	}

	return inst.TraceID, nil
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

	for _, inst := range instances {
		t.tracer.addInstance(inst)
	}
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

func (t *TracerService) addToUnconfirmedIndex(treeNode *TreeNode, utxos []*UTXO) {
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
