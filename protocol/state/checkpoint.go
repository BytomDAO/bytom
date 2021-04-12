package state

import (
	"sort"

	"github.com/bytom/bytom/protocol/bc"
)

const (
	// BlocksOfEpoch represent the block num in one epoch
	BlocksOfEpoch   = 100
	minMortgage     = 1000000
	numOfValidators = 10
)

// CheckpointStatus represent current status of checkpoint
type CheckpointStatus uint8

const (
	// Growing means that the checkpoint has not ended the current epoch
	Growing CheckpointStatus = iota

	// Unverified means thant the checkpoint has ended the current epoch, but not been justified
	Unverified

	// Justified if checkpoint is the root, or there exists a super link c′ → c where c′ is justified
	Justified

	// Finalized if checkpoint c is justified and there is a sup link c→c′ where c′is a direct child of c
	Finalized
)

// SupLink is an ordered pair of checkpoints (a, b), also written a → b,
// such that at least 2/3 of validators have published votes with source a and target b.
type SupLink struct {
	SourceHeight uint64
	SourceHash   bc.Hash
	PubKeys      map[string]bool // valid pubKeys of signature
}

// Confirmed if at least 2/3 of validators have published votes with sup link
func (s *SupLink) Confirmed() bool {
	return len(s.PubKeys) > numOfValidators*2/3
}

// Checkpoint represent the block/hash under consideration for finality for a given epoch.
// This block is the last block of the previous epoch. Rather than dealing with every block,
// Casper only considers checkpoints for finalization. When a checkpoint is explicitly finalized,
// all ancestor blocks of the checkpoint are implicitly finalized.
type Checkpoint struct {
	Height         uint64
	Hash           bc.Hash
	PrevHash       bc.Hash
	StartTimestamp uint64
	SupLinks       []*SupLink
	Status         CheckpointStatus

	Votes     map[string]uint64 // putKey -> num of vote
	Mortgages map[string]uint64 // pubKey -> num of mortgages
}

// AddSupLink add a valid sup link to checkpoint
func (c *Checkpoint) AddSupLink(sourceHeight uint64, sourceHash bc.Hash, pubKey string) *SupLink {
	for _, supLink := range c.SupLinks {
		if supLink.SourceHash == sourceHash {
			supLink.PubKeys[pubKey] = true
			return supLink
		}
	}

	supLink := &SupLink{
		SourceHeight: sourceHeight,
		SourceHash:   sourceHash,
		PubKeys:      map[string]bool{pubKey: true},
	}
	c.SupLinks = append(c.SupLinks, supLink)
	return supLink
}

// Validator represent the participants of the PoS network
// Responsible for block generation and verification
type Validator struct {
	PubKey   string
	Vote     uint64
	Mortgage uint64
}

// Validators return next epoch of validators, if the status of checkpoint is growing, return empty
func (c *Checkpoint) Validators() []*Validator {
	var validators []*Validator
	if c.Status == Growing {
		return validators
	}

	for pubKey, mortgageNum := range c.Mortgages {
		if mortgageNum >= minMortgage {
			validators = append(validators, &Validator{
				PubKey:   pubKey,
				Vote:     c.Votes[pubKey],
				Mortgage: mortgageNum,
			})
		}
	}

	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Mortgage+validators[i].Vote > validators[j].Mortgage+validators[j].Vote
	})

	end := numOfValidators
	if len(validators) < numOfValidators {
		end = len(validators)
	}
	return validators[:end]
}

// ContainsValidator check whether the checkpoint contains the pubKey as validator
func (c *Checkpoint) ContainsValidator(pubKey string) bool {
	for _, v := range c.Validators() {
		if v.PubKey == pubKey {
			return true
		}
	}
	return false
}
