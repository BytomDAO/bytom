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
	Ended
	OffChain
)

type TreeNode struct {
	TxHash   bc.Hash     `json:"tx_hash"`
	UTXOs    []*UTXO     `json:"utxos"`
	Children []*TreeNode `json:"children"`
}

type Instance struct {
	TraceID       string   `json:"trace_id"`
	UTXOs         []*UTXO  `json:"utxos"`
	TxHash        *bc.Hash `json:"tx_hash"`
	Status        Status   `json:"status"`
	EndedHeight   uint64   `json:"ended_height"`
	ScannedHash   bc.Hash  `json:"scanned_hash"`
	ScannedHeight uint64   `json:"scanned_height"`
	Unconfirmed   []*TreeNode
}

func newInstance(t *transfer, block *types.Block) *Instance {
	inst := &Instance{
		TraceID:       uuid.New().String(),
		TxHash:        &t.txHash,
		UTXOs:         t.outUTXOs,
		Status:        Lagging,
		ScannedHeight: block.Height,
		ScannedHash:   block.Hash(),
	}
	if len(t.outUTXOs) == 0 {
		inst.Status = Ended
		inst.UTXOs = t.inUTXOs
	}
	return inst
}

func (i *Instance) transferTo(t *transfer, blockHeight uint64) *Instance {
	inst := &Instance{
		TraceID:     i.TraceID,
		Status:      i.Status,
		Unconfirmed: i.Unconfirmed,
		UTXOs:       t.outUTXOs,
		TxHash:      &t.txHash,
	}
	if len(t.outUTXOs) == 0 {
		inst.Status = Ended
		inst.EndedHeight = blockHeight
		inst.UTXOs = t.inUTXOs
	}
	inst.confirmTx(t.txHash)
	return inst
}

func (i *Instance) rollbackTo(t *transfer) *Instance {
	return &Instance{
		TraceID:     i.TraceID,
		Status:      InSync,
		UTXOs:       t.inUTXOs,
		TxHash:      nil,
		Unconfirmed: nil,
	}
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

type instanceIndex struct {
	traceIdToInst  map[string]*Instance
	utxoHashToInst map[bc.Hash]*Instance
}

func newInstanceIndex() *instanceIndex {
	return &instanceIndex{
		traceIdToInst:  make(map[string]*Instance),
		utxoHashToInst: make(map[bc.Hash]*Instance),
	}
}

func (i *instanceIndex) getAll() []*Instance {
	var instances []*Instance
	for _, inst := range i.traceIdToInst {
		instances = append(instances, inst)
	}
	return instances
}

func (i *instanceIndex) getByID(traceID string) *Instance {
	return i.traceIdToInst[traceID]
}

func (i *instanceIndex) getByUTXO(utxoHash bc.Hash) *Instance {
	return i.utxoHashToInst[utxoHash]
}

func (i *instanceIndex) add(instance *Instance) {
	i.traceIdToInst[instance.TraceID] = instance
	for _, utxo := range instance.UTXOs {
		i.utxoHashToInst[utxo.Hash] = instance
	}
}

func (i *instanceIndex) save(newInst *Instance) {
	i.remove(newInst.TraceID)
	i.add(newInst)
}

func (i *instanceIndex) remove(id string) {
	if inst, ok := i.traceIdToInst[id]; ok {
		delete(i.traceIdToInst, id)
		for _, utxo := range inst.UTXOs {
			delete(i.utxoHashToInst, utxo.Hash)
		}
	}
}

type UTXO struct {
	Hash      bc.Hash    `json:"hash"`
	AssetID   bc.AssetID `json:"asset_id"`
	Amount    uint64     `json:"amount"`
	Program   []byte     `json:"program"`
	SourceID  bc.Hash    `json:"source_id"`
	SourcePos uint64     `json:"source_pos"`
	StateData [][]byte   `json:"state_data"`
}

func inputToUTXO(tx *types.Tx, index int) *UTXO {
	input := tx.Inputs[index]
	outputID, _ := input.SpentOutputID()
	spendInput := input.TypedInput.(*types.SpendInput)
	return &UTXO{
		Hash:      outputID,
		AssetID:   input.AssetID(),
		Amount:    input.Amount(),
		Program:   input.ControlProgram(),
		SourceID:  spendInput.SourceID,
		SourcePos: spendInput.SourcePosition,
		StateData: spendInput.StateData,
	}
}

func outputToUTXO(tx *types.Tx, index int) *UTXO {
	output := tx.Outputs[index]
	outputID := tx.OutputID(index)
	originalOutput, _ := tx.OriginalOutput(*outputID)
	return &UTXO{
		Hash:      *outputID,
		AssetID:   *output.AssetId,
		Amount:    output.Amount,
		Program:   output.ControlProgram,
		SourceID:  *originalOutput.Source.Ref,
		SourcePos: uint64(index),
		StateData: originalOutput.StateData,
	}
}
