package bc

import "io"

// Output is the result of a transfer of value. The value it contains
// may be accessed by a later Spend entry (if that entry can satisfy
// the Output's ControlProgram). Output satisfies the Entry interface.
//
// (Not to be confused with the deprecated type TxOutput.)

func (Output) typ() string { return "output1" }
func (o *Output) writeForHash(w io.Writer) {
	mustWriteForHash(w, o.Source)
	mustWriteForHash(w, o.ControlProgram)
}

// NewOutput creates a new Output.
func NewOutput(source *ValueSource, controlProgram *Program, ordinal uint64) *Output {
	return &Output{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
	}
}
