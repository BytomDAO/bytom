package bc

import "io"

// Spend accesses the value in a prior Output for transfer
// elsewhere. It satisfies the Entry interface.
//
// (Not to be confused with the deprecated type SpendInput.)

func (Spend) typ() string { return "spend1" }
func (s *Spend) writeForHash(w io.Writer) {
	mustWriteForHash(w, s.SpentOutputId)
}

// SetDestination will link the spend to the output
func (s *Spend) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	s.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewSpend creates a new Spend.
func NewSpend(spentOutputID *Hash, ordinal uint64) *Spend {
	return &Spend{
		SpentOutputId: spentOutputID,
		Ordinal:       ordinal,
	}
}
