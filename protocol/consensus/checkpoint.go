package consensus

import (
	"sort"
)

const (
	blocksOfEpoch   = 100
	minMortgage     = 1000000
	numOfValidators = 10
)

type checkpointStatus uint8

const (
	growing checkpointStatus = iota
	unverified
	justified
	finalized
)

type supLink struct {
	sourceHeight uint64
	sourceHash   string
	pubKeys      map[string]bool // valid pubKeys of signature
}

func (s *supLink) confirmed() bool {
	return len(s.pubKeys) > numOfValidators*2/3
}

type checkpoint struct {
	height         uint64
	hash           string
	startTimestamp uint64
	supLinks       []*supLink
	status         checkpointStatus

	votes     map[string]uint64 // putKey -> num of vote
	mortgages map[string]uint64 // pubKey -> num of mortgages
}

func (c *checkpoint) addSupLink(sourceHeight uint64, sourceHash, pubKey string) *supLink {
	for _, supLink := range c.supLinks {
		if supLink.sourceHash == sourceHash {
			supLink.pubKeys[pubKey] = true
			return supLink
		}
	}

	supLink := &supLink{
		sourceHeight: sourceHeight,
		sourceHash:   sourceHash,
		pubKeys:      map[string]bool{pubKey: true},
	}
	c.supLinks = append(c.supLinks, supLink)
	return supLink
}

// Validator represent the participants of the PoS network
// Responsible for block generation and verification
type Validator struct {
	PubKey   string
	Vote     uint64
	Mortgage uint64
}

func (c *checkpoint) validators() []*Validator {
	var validators []*Validator
	for pubKey, mortgageNum := range c.mortgages {
		if mortgageNum >= minMortgage {
			validators = append(validators, &Validator{
				PubKey:   pubKey,
				Vote:     c.votes[pubKey],
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
