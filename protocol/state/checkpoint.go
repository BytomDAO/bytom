package state

import (
	"sort"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
)

const (
	// BlocksOfEpoch represent the block num in one epoch
	BlocksOfEpoch   = 100
	minMortgage     = 1000000
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
	Signatures   [consensus.MaxNumOfValidators]string
}

// IsMajority if at least 2/3 of validators have published votes with sup link
func (s *SupLink) IsMajority(numOfValidators int) bool {
	numOfSignatures := 0
	for _, signature := range s.Signatures {
		if signature != "" {
			numOfSignatures++
		}
	}
	return numOfSignatures > numOfValidators*2/3
}

// Checkpoint represent the block/hash under consideration for finality for a given epoch.
// This block is the last block of the previous epoch. Rather than dealing with every block,
// Casper only considers checkpoints for finalization. When a checkpoint is explicitly finalized,
// all ancestor blocks of the checkpoint are implicitly finalized.
type Checkpoint struct {
	Height     uint64
	Hash       bc.Hash
	ParentHash bc.Hash
	// only save in the memory, not be persisted
	Parent    *Checkpoint `json:"-"`
	Timestamp uint64
	SupLinks  []*SupLink  `json:"-"`
	Status    CheckpointStatus

	Votes      map[string]uint64 // putKey -> num of vote
	Guaranties map[string]uint64 // pubKey -> num of guaranty
}

// AddVerification add a valid verification to checkpoint's supLink
func (c *Checkpoint) AddVerification(sourceHash bc.Hash, sourceHeight uint64, validatorOrder int, signature string) *SupLink {
	for _, supLink := range c.SupLinks {
		if supLink.SourceHash == sourceHash {
			supLink.Signatures[validatorOrder] = signature
			return supLink
		}
	}
	supLink := &SupLink{
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
		if (sourceHash == nil || supLink.SourceHash == *sourceHash) && supLink.Signatures[validatorOrder] != "" {
			return true
		}
	}
	return false
}

// Validator represent the participants of the PoS network
// Responsible for block generation and verification
type Validator struct {
	PubKey   string
	Order    int
	Vote     uint64
	Guaranty uint64
}

// Validators return next epoch of validators, if the status of checkpoint is growing, return empty
func (c *Checkpoint) Validators() map[string]*Validator {
	var validators []*Validator
	if c.Status == Growing {
		return nil
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

	if len(validators) == 0 {
		return federationValidators()
	}

	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Guaranty+validators[i].Vote > validators[j].Guaranty+validators[j].Vote
	})

	result := make(map[string]*Validator)
	for i := 0; i < len(validators) && i < consensus.MaxNumOfValidators; i++ {
		validator := validators[i]
		validator.Order = i
		result[validator.PubKey] = validator
	}
	return result
}

func federationValidators() map[string]*Validator {
	validators := map[string]*Validator{}
	for i, xPub := range config.CommonConfig.Federation.Xpubs {
		validators[xPub.String()] = &Validator{PubKey: xPub.String(), Order: i}
	}
	return validators
}
