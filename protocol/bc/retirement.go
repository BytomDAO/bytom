package bc

import "io"

// Retirement is for the permanent removal of some value from a
// blockchain. The value it contains can never be obtained by later
// entries. Retirement satisfies the Entry interface.

func (Retirement) typ() string { return "retirement1" }
func (r *Retirement) writeForHash(w io.Writer) {
	mustWriteForHash(w, r.Source)
}

// NewRetirement creates a new Retirement.
func NewRetirement(source *ValueSource, ordinal uint64) *Retirement {
	return &Retirement{
		Source:  source,
		Ordinal: ordinal,
	}
}
