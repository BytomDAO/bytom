package bc

import "io"

func (Coinbase) typ() string { return "coinbase1" }
func (n *Coinbase) writeForHash(w io.Writer) {
	mustWriteForHash(w, n.Program)
}

// NewCoinbase creates a new Coinbase.
func NewCoinbase(p *Program) *Coinbase {
	return &Coinbase{
		Program: p,
	}
}
