package contract

import (
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

type Tracer struct {
	table *InstanceTable
	infra *Infrastructure
}

func (t *Tracer) ApplyBlock(block *types.Block) error {
	return nil
}

func (t *Tracer) DetachBlock(block *types.Block) error {
	return nil
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
