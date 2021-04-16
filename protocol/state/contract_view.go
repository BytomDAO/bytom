package state

import (
	"github.com/bytom/bytom/consensus/bcrp"
	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/protocol/bc/types"
)

// ContractViewpoint represents a view into the set of registered contract
type ContractViewpoint struct {
	AttachEntries map[[32]byte][]byte
	DetachEntries map[[32]byte][]byte
}

// NewContractViewpoint returns a new empty contract view.
func NewContractViewpoint() *ContractViewpoint {
	return &ContractViewpoint{
		AttachEntries: make(map[[32]byte][]byte),
		DetachEntries: make(map[[32]byte][]byte),
	}
}

// ApplyBlock apply block contract to contract view
func (view *ContractViewpoint) ApplyBlock(block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			program := output.ControlProgram
			if !bcrp.IsBCRPScript(program) {
				continue
			}
			var hash [32]byte
			sha3pool.Sum256(hash[:], program)
			view.AttachEntries[hash] = append(tx.ID.Bytes(), program...)
		}
	}
	return nil
}

// DetachBlock detach block contract to contract view
func (view *ContractViewpoint) DetachBlock(block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			program := output.ControlProgram
			if !bcrp.IsBCRPScript(program) {
				continue
			}
			var hash [32]byte
			sha3pool.Sum256(hash[:], program)
			view.DetachEntries[hash] = append(tx.ID.Bytes(), program...)
		}
	}
	return nil
}
