package bc

import "io"

func (Coinbase) typ() string { return "coinbase1" }
func (n *Coinbase) writeForHash(w io.Writer) {
	mustWriteForHash(w, n.Program)
	mustWriteForHash(w, n.BlockId)
}

// NewCoinbase creates a new Coinbase.
func NewCoinbase(p *Program, trID *Hash) *Coinbase {
	return &Coinbase{
		Program: p,
		BlockId: trID,
	}
}
