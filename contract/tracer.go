package contract

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

type Tracer struct {
	sync.RWMutex
	table     *InstanceTable
	infra     *Infrastructure
	scheduler *TraceScheduler
}

func NewTracer(infra *Infrastructure, scheduler *TraceScheduler) *Tracer {
	instances, err := infra.Repository.LoadInstances()
	if err != nil {
		logrus.WithField("err", err).Fatal("load instances from db")
	}

	table := NewInstanceTable()
	for _, inst := range instances {
		if inst.InSync {
			table.Put(inst)
		} else {
			scheduler.AddNewJob(inst)
		}
	}
	return &Tracer{infra: infra, table: table, scheduler: scheduler}
}

func (t *Tracer) ApplyBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	var newInstances, oldInstances []*Instance
	for _, tx := range block.Transactions {
		inUTXOs, outUTXOs := t.parseTransfer(tx)
		if len(inUTXOs) == 0 {
			continue
		}

		if inst := t.table.GetByUTXO(inUTXOs[0].hash); inst != nil {
			newInst := NewInstance(inst.TraceID, inUTXOs, outUTXOs)
			newInst.InSync = true
			newInstances = append(newInstances, newInst)
			oldInstances = append(oldInstances, inst)
		}
	}
	return t.saveInstances(newInstances, oldInstances)
}

func (t *Tracer) DetachBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	var newInstances, oldInstances []*Instance
	for i := len(block.Transactions); i >= 0; i-- {
		tx := block.Transactions[i]
		inUTXOs, outUTXOs := t.parseTransfer(tx)
		utxos := append(outUTXOs, inUTXOs...)
		if len(utxos) == 0 {
			continue
		}

		if inst := t.table.GetByUTXO(utxos[0].hash); inst != nil {
			newInst := NewInstance(inst.TraceID, outUTXOs, inUTXOs)
			newInst.InSync = true
			newInstances = append(newInstances, newInst)
			oldInstances = append(oldInstances, inst)
		}
	}
	return t.saveInstances(newInstances, oldInstances)
}

func (t *Tracer) AddUnconfirmedTx(tx *types.Tx) error {
	return nil
}

func (t *Tracer) CreateInstance(txHash, blockHash bc.Hash) (string, error) {
	block, err := t.infra.Chain.GetBlock(blockHash)
	if err != nil {
		return "", err
	}

	for _, tx := range block.Transactions {
		if tx.ID == txHash {
			inUTXOs, outUTXOs := t.parseTransfer(tx)
			if len(inUTXOs) == 0 {
				return "", errors.New("input of tx has not contract")
			}

			inst := NewInstance(uuid.New().String(), inUTXOs, outUTXOs)
			if err := t.infra.Repository.SaveInstances([]*Instance{inst}); err != nil {
				return "", err
			}

			if !inst.Finalized {
				t.scheduler.AddNewJob(inst)
			}
			return inst.TraceID, nil
		}
	}
	return "", errors.New("tx hash and block hash is mismatch")
}

func (t *Tracer) RemoveInstance(traceID string) error {
	return nil
}

func (t *Tracer) GetInstance(traceID string) (*Instance, error) {
	return nil, nil
}

func (t *Tracer) takeOverInstance(instance *Instance) bool {
	return false
}

func (t *Tracer) parseTransfer(tx *types.Tx) ([]*UTXO, []*UTXO) {
	var inUTXOs, outUTXOs []*UTXO
	for i, input := range tx.Inputs {
		if segwit.IsP2WSHScript(input.ControlProgram()) {
			inUTXOs = append(inUTXOs, inputToUTXO(tx, i))
		}
	}

	for i, output := range tx.Outputs {
		if segwit.IsP2WSHScript(output.ControlProgram) {
			outUTXOs = append(outUTXOs, outputToUTXO(tx, i))
		}
	}
	return inUTXOs, outUTXOs
}

func (t *Tracer) saveInstances(instances, oldInstances []*Instance) error {
	if err := t.infra.Repository.SaveInstances(instances); err != nil {
		return err
	}

	for _, inst := range oldInstances {
		t.table.Remove(inst.TraceID)
	}

	for _, inst := range instances {
		t.table.Put(inst)
	}
	return nil
}
