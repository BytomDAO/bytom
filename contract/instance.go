package contract

import (
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/google/uuid"
)

type Status uint8

const (
	Lagging Status = iota + 1
	InSync
	Finalized
	OffChain
)

type treeNode struct {
	TxHash   bc.Hash
	UTXOs    []*UTXO
	Children []*treeNode
}

type Instance struct {
	TraceID       string
	UTXOs         []*UTXO
	Unconfirmed   []*treeNode
	Status        Status
	ScannedHash   bc.Hash
	ScannedHeight uint64
}

func newInstance(inUTXOs, outUTXOs []*UTXO) *Instance {
	inst := &Instance{
		TraceID: uuid.New().String(),
		UTXOs:   outUTXOs,
	}
	inst.Status = Lagging
	if len(outUTXOs) == 0 {
		inst.Status = Finalized
		inst.UTXOs = inUTXOs
	}
	return inst
}

func (i *Instance) transferTo(newUTXOs []*UTXO) *Instance {
	inst := &Instance{
		TraceID:     i.TraceID,
		Status:      i.Status,
		Unconfirmed: i.Unconfirmed,
		UTXOs:       newUTXOs,
	}
	if len(newUTXOs) == 0 {
		inst.Status = Finalized
		inst.UTXOs = i.UTXOs
	}
	return inst
}

func (i *Instance) confirmTx(txHash bc.Hash) {
	for _, node := range i.Unconfirmed {
		if node.TxHash == txHash {
			i.Unconfirmed = node.Children
			return
		}
	}
	i.Unconfirmed = nil
}

type instanceTable struct {
	traceIdToInst  map[string]*Instance
	utxoHashToInst map[bc.Hash]*Instance
}

func newInstanceTable() *instanceTable {
	return &instanceTable{
		traceIdToInst:  make(map[string]*Instance),
		utxoHashToInst: make(map[bc.Hash]*Instance),
	}
}

func (i *instanceTable) getByID(id string) *Instance {
	return i.traceIdToInst[id]
}

func (i *instanceTable) getByUTXO(utxoHash bc.Hash) *Instance {
	return i.utxoHashToInst[utxoHash]
}

func (i *instanceTable) save(newInst *Instance) {
	if old, ok := i.traceIdToInst[newInst.TraceID]; ok {
		for _, utxo := range old.UTXOs {
			delete(i.utxoHashToInst, utxo.hash)
		}
	}
	i.add(newInst)
}

func (i *instanceTable) remove(id string) {
	if inst, ok := i.traceIdToInst[id]; ok {
		delete(i.traceIdToInst, id)
		for _, utxo := range inst.UTXOs {
			delete(i.utxoHashToInst, utxo.hash)
		}
	}
}

func (i *instanceTable) add(instance *Instance) {
	i.traceIdToInst[instance.TraceID] = instance
	for _, utxo := range instance.UTXOs {
		i.utxoHashToInst[utxo.hash] = instance
	}
}

type UTXO struct {
	hash      bc.Hash
	assetID   bc.AssetID
	amount    uint64
	program   []byte
	sourceID  bc.Hash
	sourcePos uint64
	stateData [][]byte
}

func inputToUTXO(tx *types.Tx, index int) *UTXO {
	input := tx.Inputs[index]
	spendInput := input.TypedInput.(*types.SpendInput)
	return &UTXO{
		hash:      tx.InputIDs[index],
		assetID:   input.AssetID(),
		amount:    input.Amount(),
		program:   input.ControlProgram(),
		sourceID:  spendInput.SourceID,
		sourcePos: spendInput.SourcePosition,
		stateData: spendInput.StateData,
	}
}

func outputToUTXO(tx *types.Tx, index int) *UTXO {
	output := tx.Outputs[index]
	outputID := tx.OutputID(index)
	originalOutput, _ := tx.OriginalOutput(*outputID)
	return &UTXO{
		hash:      *outputID,
		assetID:   *output.AssetId,
		amount:    output.Amount,
		program:   output.ControlProgram,
		sourceID:  *originalOutput.Source.Ref,
		sourcePos: uint64(index),
		stateData: originalOutput.StateData,
	}
}
