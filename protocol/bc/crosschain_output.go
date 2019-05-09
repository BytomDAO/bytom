package bc

import "io"

func (CrossChainOutput) typ() string { return "crosschainoutput1" }
func (o *CrossChainOutput) writeForHash(w io.Writer) {
	mustWriteForHash(w, o.Source)
	mustWriteForHash(w, o.ControlProgram)
}

// NewClaimOutput creates a new CrossChainOutput for a claim tx.
func NewCrossChainOutput(source *ValueSource, controlProgram *Program, ordinal uint64) *CrossChainOutput {
	return &CrossChainOutput{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
	}
}
