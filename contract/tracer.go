package contract

import (
	"sync"

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

	var instances []*Instance
	for _, tx := range block.Transactions {
		inUTXOs, outUTXOs := t.parseTransfer(tx)
		if len(inUTXOs) == 0 {
			continue
		}

		if inst := t.table.GetByUTXO(inUTXOs[0].hash); inst != nil {
			newInst := NewInstance(inUTXOs, outUTXOs)
			newInst.TraceID = inst.TraceID
			newInst.InSync = true
			instances = append(instances, newInst)
		}
	}
	return t.saveInstances(instances)
}

func (t *Tracer) DetachBlock(block *types.Block) error {
	t.Lock()
	defer t.Unlock()

	var instances []*Instance
	for _, tx := range block.Transactions {
		inUTXOs, outUTXOs := t.parseTransfer(tx)
		utxos := outUTXOs
		if len(outUTXOs) == 0 {
			utxos = inUTXOs
		}
		if len(utxos) == 0 {
			continue
		}

		if inst := t.table.GetByUTXO(utxos[0].hash); inst != nil {
			instances = append(instances, &Instance{
				TraceID:   inst.TraceID,
				UTXOs:     inUTXOs,
				Finalized: false,
				InSync:    true,
			})
		}
	}
	return t.saveInstances(instances)
}

func (t *Tracer) AddUnconfirmedTx(tx *types.Tx) error {
	return nil
}

func (t *Tracer) CreateInstance(txHash, blockHash bc.Hash) (string, error) {
	return "", nil
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

func (t *Tracer) saveInstances(instances []*Instance) error {
	if err := t.infra.Repository.SaveInstances(instances); err != nil {
		return err
	}

	for _, inst := range instances {
		t.table.Put(inst)
	}
	return nil
}
