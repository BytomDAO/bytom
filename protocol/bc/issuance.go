package bc

import "io"

// Issuance is a source of new value on a blockchain. It satisfies the
// Entry interface.

func (Issuance) typ() string { return "issuance1" }
func (iss *Issuance) writeForHash(w io.Writer) {
	mustWriteForHash(w, iss.NonceHash)
	mustWriteForHash(w, iss.Value)
}

// SetDestination will link the issuance to the output
func (iss *Issuance) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	iss.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewIssuance creates a new Issuance.
func NewIssuance(nonceHash *Hash, value *AssetAmount, ordinal uint64) *Issuance {
	return &Issuance{
		NonceHash: nonceHash,
		Value:     value,
		Ordinal:   ordinal,
	}
}
