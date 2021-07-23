package bc

import "io"

// OriginalOutput is the result of a transfer of value. The value it contains
// may be accessed by a later Spend entry (if that entry can satisfy
// the Output's ControlProgram). OriginalOutput satisfies the Entry interface.

func (OriginalOutput) typ() string { return "originalOutput1" }
func (o *OriginalOutput) writeForHash(w io.Writer) {
	mustWriteForHash(w, o.Source)
	mustWriteForHash(w, o.ControlProgram)
	mustWriteForHash(w, o.StateData)
}

// NewOriginalOutput creates a new OriginalOutput.
func NewOriginalOutput(source *ValueSource, controlProgram *Program, stateData [][]byte, ordinal uint64) *OriginalOutput {
	return &OriginalOutput{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
		StateData:      stateData,
	}
}
