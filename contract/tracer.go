package contract

import (
	"encoding/hex"

	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

type tracer struct {
	index *instanceIndex
}

func newTracer(instances []*Instance) *tracer {
	index := newInstanceIndex()
	for _, inst := range instances {
		index.save(inst)
	}
	return &tracer{index: index}
}

func (t *tracer) getInstance(traceID string) *Instance {
	return t.index.getByID(traceID)
}

func (t *tracer) allInstances() []*Instance {
	return t.index.getAll()
}

func (t *tracer) addInstances(instances []*Instance) {
	for _, inst := range instances {
		t.index.save(inst)
	}
}

func (t *tracer) removeInstance(traceID string) {
	t.index.remove(traceID)
}

func (t *tracer) applyBlock(block *types.Block) []*Instance {
	var newInstances []*Instance
	for _, tx := range block.Transactions {
		transfers := parseTransfers(tx)
		for _, transfer := range transfers {
			if len(transfer.inUTXOs) == 0 {
				continue
			}

			if inst := t.index.getByUTXO(transfer.inUTXOs[0].Hash); inst != nil {
				newInst := inst.transferTo(transfer, block.Height)
				newInstances = append(newInstances, newInst)
			}
		}
	}
	t.saveInstances(newInstances)
	return newInstances
}

func (t *tracer) detachBlock(block *types.Block) []*Instance {
	var newInstances []*Instance
	for i := len(block.Transactions) - 1; i >= 0; i-- {
		tx := block.Transactions[i]
		transfers := parseTransfers(tx)
		for _, transfer := range transfers {
			utxos := append(transfer.outUTXOs, transfer.inUTXOs...)
			if inst := t.index.getByUTXO(utxos[0].Hash); inst != nil {
				newInst := inst.rollbackTo(transfer)
				newInstances = append(newInstances, newInst)
			}
		}
	}
	t.saveInstances(newInstances)
	return newInstances
}

type transfer struct {
	txHash   bc.Hash
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
		transfers = append(transfers, &transfer{txHash: tx.ID, inUTXOs: utxos, outUTXOs: outUTXOs})
	}
	for program, utxos := range groupOutUTXOs {
		if _, ok := groupInUTXOs[program]; !ok {
			transfers = append(transfers, &transfer{txHash: tx.ID, outUTXOs: utxos})
		}
	}
	return transfers
}

func groupUTXOs(utxos []*UTXO) map[string][]*UTXO {
	groupUTXOs := make(map[string][]*UTXO)
	for _, utxo := range utxos {
		program := hex.EncodeToString(utxo.Program)
		groupUTXOs[program] = append(groupUTXOs[program], utxo)
	}
	return groupUTXOs
}

func parseContractUTXOs(tx *types.Tx) ([]*UTXO, []*UTXO) {
	var inUTXOs, outUTXOs []*UTXO
	for i, input := range tx.Inputs {
		if isContract(input.ControlProgram()) && input.InputType() == types.SpendInputType {
			inUTXOs = append(inUTXOs, inputToUTXO(tx, i))
		}
	}

	for i, output := range tx.Outputs {
		if isContract(output.ControlProgram) && output.OutputType() == types.OriginalOutputType {
			outUTXOs = append(outUTXOs, outputToUTXO(tx, i))
		}
	}
	return inUTXOs, outUTXOs
}

func isContract(program []byte) bool {
	return !(segwit.IsP2WScript(program) || vmutil.IsUnspendable(program))
}

func (t *tracer) saveInstances(instances []*Instance) {
	for _, inst := range instances {
		t.index.save(inst)
	}
}
