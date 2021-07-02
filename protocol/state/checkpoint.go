package state

import (
	"encoding/hex"
	"sort"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

// CheckpointStatus represent current status of checkpoint
type CheckpointStatus uint8

const (
	// Growing means that the checkpoint has not ended the current epoch
	Growing CheckpointStatus = iota

	// Unjustified means thant the checkpoint has ended the current epoch, but not been justified
	Unjustified

	// Justified if checkpoint is the root, or there exists a super link c′ → c where c′ is justified
	Justified

	// Finalized if checkpoint c is justified and there is a sup link c→c′ where c′is a direct child of c
	Finalized
)

var errIncreaseCheckpoint = errors.New("invalid block for increase checkpoint")

// Checkpoint represent the block/hash under consideration for finality for a given epoch.
// This block is the last block of the previous epoch. Rather than dealing with every block,
// Casper only considers checkpoints for finalization. When a checkpoint is explicitly finalized,
// all ancestor blocks of the checkpoint are implicitly finalized.
type Checkpoint struct {
	Height     uint64
	Hash       bc.Hash
	ParentHash bc.Hash
	Timestamp  uint64
	Status     CheckpointStatus

	Rewards map[string]uint64 // controlProgram -> num of reward
	Votes   map[string]uint64 // pubKey -> num of vote

	// only save in the memory, not be persisted
	Parent   *Checkpoint      `json:"-"`
	SupLinks []*types.SupLink `json:"-"`
}

// NewCheckpoint create a new checkpoint instance
func NewCheckpoint(parent *Checkpoint) *Checkpoint {
	checkpoint := &Checkpoint{
		ParentHash: parent.Hash,
		Parent:     parent,
		Status:     Growing,
		Rewards:    make(map[string]uint64),
		Votes:      make(map[string]uint64),
	}

	for pubKey, num := range parent.Votes {
		if num != 0 {
			checkpoint.Votes[pubKey] = num
		}
	}
	return checkpoint
}

// AddVerification add a valid verification to checkpoint's supLink
func (c *Checkpoint) AddVerification(sourceHash bc.Hash, sourceHeight uint64, validatorOrder int, signature []byte) *types.SupLink {
	for _, supLink := range c.SupLinks {
		if supLink.SourceHash == sourceHash {
			supLink.Signatures[validatorOrder] = signature
			return supLink
		}
	}

	supLink := &types.SupLink{
		SourceHeight: sourceHeight,
		SourceHash:   sourceHash,
	}
	supLink.Signatures[validatorOrder] = signature
	c.SupLinks = append(c.SupLinks, supLink)
	return supLink
}

// ContainsVerification return whether the specified validator has add verification to current checkpoint
// sourceHash not as filter if is nil,
func (c *Checkpoint) ContainsVerification(validatorOrder int, sourceHash *bc.Hash) bool {
	for _, supLink := range c.SupLinks {
		if (sourceHash == nil || supLink.SourceHash == *sourceHash) && len(supLink.Signatures[validatorOrder]) != 0 {
			return true
		}
	}
	return false
}

// Increase will increase the height of checkpoint
func (c *Checkpoint) Increase(block *types.Block) error {
	empty := bc.Hash{}
	prevHash := c.Hash
	if c.Hash == empty {
		prevHash = c.ParentHash
	}

	if block.PreviousBlockHash != prevHash {
		return errIncreaseCheckpoint
	}

	c.Hash = block.Hash()
	c.Height = block.Height
	c.Timestamp = block.Timestamp
	return nil
}

// Validator represent the participants of the PoS network
// Responsible for block generation and verification
type Validator struct {
	PubKey  string
	Order   int
	VoteNum uint64
}

// EffectiveValidators return next epoch of effective validators, if the status of checkpoint is growing, return empty
func (c *Checkpoint) EffectiveValidators() map[string]*Validator {
	validators := c.AllValidators()
	if len(validators) == 0 {
		return federationValidators()
	}

	result := make(map[string]*Validator)
	for i := 0; i < len(validators) && i < consensus.MaxNumOfValidators; i++ {
		validator := validators[i]
		validator.Order = i
		result[validator.PubKey] = validator
	}
	return result
}

// AllValidators return all validators has vote num
func (c *Checkpoint) AllValidators() []*Validator {
	if c.Status == Growing {
		return nil
	}

	var validators []*Validator
	for pubKey, voteNum := range c.Votes {
		if voteNum >= consensus.ActiveNetParams.MinValidatorVoteNum {
			validators = append(validators, &Validator{
				PubKey:  pubKey,
				VoteNum: c.Votes[pubKey],
			})
		}
	}

	sort.Slice(validators, func(i, j int) bool {
		numI, numJ := validators[i].VoteNum, validators[j].VoteNum
		if numI != numJ {
			return numI > numJ
		}
		return validators[i].PubKey > validators[j].PubKey
	})
	return validators
}

func (c *Checkpoint) ApplyVotes(block *types.Block) {
	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if vetoInput, ok := input.TypedInput.(*types.VetoInput); ok {
				pubKey := hex.EncodeToString(vetoInput.Vote)
				if c.Votes[pubKey] > vetoInput.Amount {
					c.Votes[pubKey] -= vetoInput.Amount
				} else {
					delete(c.Votes, pubKey)
				}
			}
		}

		for _, output := range tx.Outputs {
			if voteOutput, ok := output.TypedOutput.(*types.VoteOutput); ok {
				c.Votes[hex.EncodeToString(voteOutput.Vote)] += output.Amount
			}
		}
	}
}

func federationValidators() map[string]*Validator {
	validators := map[string]*Validator{}
	for i, xPub := range config.CommonConfig.Federation.Xpubs {
		validators[xPub.String()] = &Validator{PubKey: xPub.String(), Order: i}
	}
	return validators
}
