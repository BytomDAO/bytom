package types

import (
	"github.com/bytom/vapor/protocol/bc"
)

// VoteOutput satisfies the TypedOutput interface and represents a vote transaction.
type VoteOutput struct {
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
	Vote             []byte
}

// NewVoteOutput create a new output struct
func NewVoteOutput(assetID bc.AssetID, amount uint64, controlProgram []byte, vote []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		TypedOutput: &VoteOutput{
			OutputCommitment: OutputCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &assetID,
					Amount:  amount,
				},
				VMVersion:      1,
				ControlProgram: controlProgram,
			},
			Vote: vote,
		},
	}
}

// OutputType implement the txout interface
func (it *VoteOutput) OutputType() uint8 { return VoteOutputType }
