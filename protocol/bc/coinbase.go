package bc

import "io"

func (Coinbase) typ() string { return "coinbase1" }
func (c *Coinbase) writeForHash(w io.Writer) {
	mustWriteForHash(w, c.Arbitrary)
}

// SetDestination is support function for map tx
func (c *Coinbase) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	c.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewCoinbase creates a new Coinbase.
func NewCoinbase(arbitrary []byte) *Coinbase {
	return &Coinbase{Arbitrary: arbitrary}
}
