package contract

import (
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

type Status uint8

const (
	Lagging Status = iota + 1
	InSync
	Finalized
	OffChain
)

type TreeNode struct {
	TxHash   bc.Hash
	UTXOs    []*UTXO
	Children []*TreeNode
}

type Instance struct {
	TraceID       string
	UTXOs         []*UTXO
	Unconfirmed   []*TreeNode
	Status        Status
	ScannedHash   bc.Hash
	ScannedHeight uint64
}

func NewInstance(traceID string, inUTXOs, outUTXOs []*UTXO) *Instance {
	inst := &Instance{
		TraceID: traceID,
		UTXOs:   outUTXOs,
	}
	inst.Status = Lagging
	if len(outUTXOs) == 0 {
		inst.Status = Finalized
		inst.UTXOs = inUTXOs
	}
	return inst
}

type InstanceTable struct {
	traceIdToInst  map[string]*Instance
	utxoHashToInst map[bc.Hash]*Instance
}

func NewInstanceTable() *InstanceTable {
	return &InstanceTable{
		traceIdToInst:  make(map[string]*Instance),
		utxoHashToInst: make(map[bc.Hash]*Instance),
	}
}

func (i *InstanceTable) GetByID(id string) *Instance {
	return i.traceIdToInst[id]
}

func (i *InstanceTable) GetByUTXO(utxoHash bc.Hash) *Instance {
	return i.utxoHashToInst[utxoHash]
}

func (i *InstanceTable) Put(instance *Instance) {
	i.traceIdToInst[instance.TraceID] = instance
	for _, utxo := range instance.UTXOs {
		i.utxoHashToInst[utxo.hash] = instance
	}
}

func (i *InstanceTable) Remove(id string) {
	if inst, ok := i.traceIdToInst[id]; ok {
		delete(i.traceIdToInst, id)
		for _, utxo := range inst.UTXOs {
			delete(i.utxoHashToInst, utxo.hash)
		}
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
