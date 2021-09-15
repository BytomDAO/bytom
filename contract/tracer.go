package contract

import (
	"encoding/hex"

	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/protocol/bc/types"
)

type tracer struct {
	table *instanceTable
}

func newTracer(instances []*Instance) *tracer {
	table := newInstanceTable()
	for _, inst := range instances {
		table.save(inst)
	}
	return &tracer{table: table}
}

func (t *tracer) addInstances(instances []*Instance) {
	for _, inst := range instances {
		t.table.save(inst)
	}
}

func (t *tracer) removeInstance(traceID string) {
	t.table.remove(traceID)
}

func (t *tracer) applyBlock(block *types.Block) []*Instance {
	var newInstances []*Instance
	for _, tx := range block.Transactions {
		transfers := parseTransfers(tx)
		for _, transfer := range transfers {
			if len(transfer.inUTXOs) == 0 {
				continue
			}

			if inst := t.table.getByUTXO(transfer.inUTXOs[0].hash); inst != nil {
				newInst := inst.transferTo(transfer.outUTXOs)
				newInst.confirmTx(tx.ID)
				newInstances = append(newInstances, newInst)
			}
		}
	}
	t.saveInstances(newInstances)
	return newInstances
}

func (t *tracer) detachBlock(block *types.Block) []*Instance {
	var newInstances []*Instance
	for i := len(block.Transactions); i >= 0; i-- {
		tx := block.Transactions[i]
		transfers := parseTransfers(tx)
		for _, transfer := range transfers {
			utxos := append(transfer.outUTXOs, transfer.inUTXOs...)
			if inst := t.table.getByUTXO(utxos[0].hash); inst != nil {
				newInst := inst.transferTo(transfer.inUTXOs)
				newInst.Unconfirmed = nil
				newInstances = append(newInstances, newInst)
			}
		}
	}
	t.saveInstances(newInstances)
	return newInstances
}

type transfer struct {
	inUTXOs  []*UTXO
	outUTXOs []*UTXO
}

func parseTransfers(tx *types.Tx) []*transfer {
	inUTXOs, outUTXOs := parseContractUTXOs(tx)
	groupInUTXOs := groupUTXOs(inUTXOs)
	groupOutUTXOs := groupUTXOs(outUTXOs)

	var transfers []*transfer
	for program, utxos := range groupInUTXOs {
		outUTXOs := groupOutUTXOs[program]
		transfers = append(transfers, &transfer{inUTXOs: utxos, outUTXOs: outUTXOs})
	}
	for program, utxos := range groupOutUTXOs {
		if _, ok := groupInUTXOs[program]; !ok {
			transfers = append(transfers, &transfer{outUTXOs: utxos})
		}
	}
	return transfers
}

func groupUTXOs(utxos []*UTXO) map[string][]*UTXO {
	groupUTXOs := make(map[string][]*UTXO)
	for _, utxo := range utxos {
		program := hex.EncodeToString(utxo.program)
		groupUTXOs[program] = append(groupUTXOs[program], utxo)
	}
	return groupUTXOs
}

func parseContractUTXOs(tx *types.Tx) ([]*UTXO, []*UTXO) {
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

func (t *tracer) saveInstances(instances []*Instance) {
	for _, inst := range instances {
		t.table.save(inst)
	}
}
