package bc

import "io"

func (VoteOutput) typ() string { return "voteoutput1" }
func (o *VoteOutput) writeForHash(w io.Writer) {
	mustWriteForHash(w, o.Source)
	mustWriteForHash(w, o.ControlProgram)
	mustWriteForHash(w, o.Vote)
}

// NewCrossChainOutput creates a new CrossChainOutput.
func NewVoteOutput(source *ValueSource, controlProgram *Program, ordinal uint64, vote []byte) *VoteOutput {
	return &VoteOutput{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
		Vote:           vote,
	}
}
