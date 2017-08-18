package bc

import "io"

func (Coinbase) typ() string { return "coinbase1" }
func (c *Coinbase) writeForHash(w io.Writer) {
	mustWriteForHash(w, c.WitnessDestination)
}

// NewCoinbase creates a new Coinbase.
func NewCoinbase(id *Hash, val *AssetAmount, pos uint64) *Coinbase {
	return &Coinbase{
		WitnessDestination: &ValueDestination{
			Ref:      id,
			Value:    val,
			Position: pos,
		},
	}
}
