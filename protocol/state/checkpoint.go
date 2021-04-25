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

	// Unjustified means thant the checkpoint has ended the current epoch, but not been justified
	Unjustified

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
	Signatures   map[string]string // pubKey to signature
}

// IsMajority if at least 2/3 of validators have published votes with sup link
func (s *SupLink) IsMajority() bool {
	return len(s.Signatures) > numOfValidators*2/3
}

// Checkpoint represent the block/hash under consideration for finality for a given epoch.
// This block is the last block of the previous epoch. Rather than dealing with every block,
// Casper only considers checkpoints for finalization. When a checkpoint is explicitly finalized,
// all ancestor blocks of the checkpoint are implicitly finalized.
type Checkpoint struct {
	Height         uint64
	Hash           bc.Hash
	ParentHash     bc.Hash
	// only save in the memory, not be persisted
	Parent         *Checkpoint
	StartTimestamp uint64
	SupLinks       []*SupLink
	Status         CheckpointStatus

	Votes      map[string]uint64 // putKey -> num of vote
	Guaranties map[string]uint64 // pubKey -> num of guaranty
}

// AddVerification add a valid verification to checkpoint's supLink, return the one
func (c *Checkpoint) AddVerification(sourceHash bc.Hash, sourceHeight uint64, pubKey, signature string) *SupLink {
	for _, supLink := range c.SupLinks {
		if supLink.SourceHash == sourceHash {
			supLink.Signatures[pubKey] = signature
			return supLink
		}
	}
	supLink := &SupLink{
		SourceHeight: sourceHeight,
		SourceHash:   sourceHash,
		Signatures:   map[string]string{pubKey: signature},
	}
	c.SupLinks = append(c.SupLinks, supLink)
	return supLink
}

// Validator represent the participants of the PoS network
// Responsible for block generation and verification
type Validator struct {
	PubKey   string
	Vote     uint64
	Guaranty uint64
}

// Validators return next epoch of validators, if the status of checkpoint is growing, return empty
func (c *Checkpoint) Validators() []*Validator {
	var validators []*Validator
	if c.Status == Growing {
		return validators
	}

	for pubKey, mortgageNum := range c.Guaranties {
		if mortgageNum >= minMortgage {
			validators = append(validators, &Validator{
				PubKey:   pubKey,
				Vote:     c.Votes[pubKey],
				Guaranty: mortgageNum,
			})
		}
	}

	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Guaranty+validators[i].Vote > validators[j].Guaranty+validators[j].Vote
	})

	end := numOfValidators
	if len(validators) < numOfValidators {
		end = len(validators)
	}
	return validators[:end]
}
