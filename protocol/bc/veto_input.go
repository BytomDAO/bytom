package bc

import "io"

func (VetoInput) typ() string { return "vetoInput1" }
func (s *VetoInput) writeForHash(w io.Writer) {
	mustWriteForHash(w, s.SpentOutputId)
}

// SetDestination will link the spend to the output
func (s *VetoInput) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	s.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewVetoInput creates a new VetoInput.
func NewVetoInput(spentOutputID *Hash, ordinal uint64) *VetoInput {
	return &VetoInput{
		SpentOutputId: spentOutputID,
		Ordinal:       ordinal,
	}
}
