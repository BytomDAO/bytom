package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
)

// VoteOutput satisfies the TypedOutput interface and represents a vote transaction.
type VoteOutput struct {
	Vote []byte
}

// NewVoteOutput create a new output struct
func NewVoteOutput(assetID bc.AssetID, amount uint64, controlProgram []byte, vote []byte, state [][]byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
			StateData:      state,
		},
		TypedOutput: &VoteOutput{Vote: vote},
	}
}

func (v *VoteOutput) readFrom(r *blockchain.Reader) error {
	var err error
	if v.Vote, err = blockchain.ReadVarstr31(r); err != nil {
		return errors.Wrap(err, "reading vote output vote")
	}
	return nil
}

func (v *VoteOutput) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstr31(w, v.Vote)
	return err
}

// OutputType implement the txout interface
func (v *VoteOutput) OutputType() uint8 { return VoteOutputType }
