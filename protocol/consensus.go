package protocol

import (
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

// Casper is BFT based proof of stack consensus algorithm, it provides safety and liveness in theory
type CasperConsensus interface {

	// Best chain return the chain containing the justified checkpoint of the largest height
	BestChain() (uint64, bc.Hash)

	// LastFinalized return the block height and block hash which is finalized ast last
	LastFinalized() (uint64, bc.Hash)

	// AuthVerification verify whether the Verification is legal.
	AuthVerification(v *Verification) error

	// ApplyBlock apply block to the consensus module
	ApplyBlock(block *types.Block) (*Verification, error)

	// Validators return the validators by specified block hash
	Validators(blockHash *bc.Hash) ([]*state.Validator, error)
}
