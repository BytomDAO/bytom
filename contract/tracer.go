package contract

import (
	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/protocol/bc/types"
)

type tracer struct {
	table *InstanceTable
}

func newTracer(instances []*Instance) *tracer {
	table := NewInstanceTable()
	for _, inst := range instances {
		table.Put(inst)
	}
	return &tracer{table: table}
}

func (t *tracer) addInstances(instances []*Instance) {
	for _, inst := range instances {
		t.table.Put(inst)
	}
}

func (t *tracer) addInstance(instance *Instance) {
	t.table.Put(instance)
}

func (t *tracer) removeInstance(traceID string) {
	t.table.Remove(traceID)
}

func (t *tracer) applyBlock(block *types.Block) []*Instance {
	var newInstances, oldInstances []*Instance
	for _, tx := range block.Transactions {
		inUTXOs, outUTXOs := parseTransfer(tx)
		if len(inUTXOs) == 0 {
			continue
		}

		if inst := t.table.GetByUTXO(inUTXOs[0].hash); inst != nil {
			newInst := NewInstance(inst.TraceID, inUTXOs, outUTXOs)
			newInst.Status = InSync
			for _, node := range inst.Unconfirmed {
				if node.TxHash == tx.ID {
					newInst.Unconfirmed = node.Children
				}
			}
			newInstances = append(newInstances, newInst)
			oldInstances = append(oldInstances, inst)
		}
	}
	t.saveInstances(newInstances, oldInstances)
	return newInstances
}

func (t *tracer) detachBlock(block *types.Block) []*Instance {
	var newInstances, oldInstances []*Instance
	for i := len(block.Transactions); i >= 0; i-- {
		tx := block.Transactions[i]
		inUTXOs, outUTXOs := parseTransfer(tx)
		utxos := append(outUTXOs, inUTXOs...)
		if len(utxos) == 0 {
			continue
		}

		if inst := t.table.GetByUTXO(utxos[0].hash); inst != nil {
			newInst := NewInstance(inst.TraceID, outUTXOs, inUTXOs)
			newInst.Status = InSync
			newInstances = append(newInstances, newInst)
			oldInstances = append(oldInstances, inst)
		}
	}
	t.saveInstances(newInstances, oldInstances)
	return newInstances
}

func parseTransfer(tx *types.Tx) ([]*UTXO, []*UTXO) {
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

func (t *tracer) saveInstances(instances, oldInstances []*Instance) {
	for _, inst := range oldInstances {
		t.table.Remove(inst.TraceID)
	}

	for _, inst := range instances {
		t.table.Put(inst)
	}
}
