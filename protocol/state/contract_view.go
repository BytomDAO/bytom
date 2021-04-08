package state

import (
	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/protocol/bc/types"
)

// ContractViewpoint represents a view into the set of registered contract
type ContractViewpoint struct {
	Entries map[[32]byte][]byte
}

// NewContractViewpoint returns a new empty contract view.
func NewContractViewpoint() *ContractViewpoint {
	return &ContractViewpoint{
		Entries: make(map[[32]byte][]byte),
	}
}

// ProcessBlock process block registered contract to contract view
func (view *ContractViewpoint) ProcessBlock(block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			program := output.ControlProgram
			if !segwit.IsBCRPScript(program) {
				continue
			}
			var hash [32]byte
			sha3pool.Sum256(hash[:], program)
			view.Entries[hash] = append(tx.ID.Bytes(), program...)
		}
	}
	return nil
}
