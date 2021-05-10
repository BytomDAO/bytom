package protocol

import (
	"encoding/hex"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/math/checked"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

// ApplyBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
// it will return verification when an epoch is reached and the current node is the validator, otherwise return nil
// the chain module must broadcast the verification
func (c *Casper) ApplyBlock(block *types.Block) (*Verification, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.tree.nodeByHash(block.Hash()); err == nil {
		// already processed
		return nil, nil
	}

	target, err := c.applyBlockToCheckpoint(block)
	if err != nil {
		return nil, errors.Wrap(err, "apply block to checkpoint")
	}

	if err := c.applyTransactions(target, block.Transactions); err != nil {
		return nil, err
	}

	validators, err := c.Validators(&target.Hash)
	if err != nil {
		return nil, err
	}

	if err := c.applySupLinks(target, block.SupLinks, validators); err != nil {
		return nil, err
	}

	if block.Height%state.BlocksOfEpoch == 0 {
		c.newEpochCh <- block.Hash()
	}

	return c.myVerification(target, validators)
}

func (c *Casper) applyBlockToCheckpoint(block *types.Block) (*state.Checkpoint, error) {
	node, err := c.tree.nodeByHash(block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	checkpoint := node.checkpoint
	if mod := block.Height % state.BlocksOfEpoch; mod == 1 {
		parent := checkpoint
		checkpoint = &state.Checkpoint{
			ParentHash:     parent.Hash,
			Parent:         parent,
			StartTimestamp: block.Timestamp,
			Status:         state.Growing,
			Votes:          make(map[string]uint64),
			Guaranties:     make(map[string]uint64),
		}
		node.children = append(node.children, &treeNode{checkpoint: checkpoint})
	} else if mod == 0 {
		checkpoint.Status = state.Unjustified
	}

	checkpoint.Height = block.Height
	checkpoint.Hash = block.Hash()
	return checkpoint, nil
}

func (c *Casper) applyTransactions(target *state.Checkpoint, transactions []*types.Tx) error {
	for _, tx := range transactions {
		for _, input := range tx.Inputs {
			if vetoInput, ok := input.TypedInput.(*types.VetoInput); ok {
				if err := processVeto(vetoInput, target); err != nil {
					return err
				}
			}

			if isGuarantyProgram(input.ControlProgram()) {
				if err := processWithdrawal(decodeGuarantyArgs(input.ControlProgram()), target); err != nil {
					return err
				}
			}
		}

		for _, output := range tx.Outputs {
			if _, ok := output.TypedOutput.(*types.VoteOutput); ok {
				if err := processVote(output, target); err != nil {
					return err
				}
			}

			if isGuarantyProgram(output.ControlProgram) {
				if err := processGuaranty(decodeGuarantyArgs(output.ControlProgram), target); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// applySupLinks copy the block's supLink to the checkpoint
func (c *Casper) applySupLinks(target *state.Checkpoint, supLinks []*types.SupLink, validators []*state.Validator) error {
	if target.Height%state.BlocksOfEpoch != 0 {
		return nil
	}

	for _, supLink := range supLinks {
		for _, verification := range supLinkToVerifications(supLink, validators, target.Hash, target.Height) {
			if err := c.verifyVerification(verification, true); err == nil {
				if err := c.addVerificationToCheckpoint(target, verification); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Casper) myVerification(target *state.Checkpoint, validators []*state.Validator) (*Verification, error) {
	pubKey := config.CommonConfig.PrivateKey().XPub().String()
	if !isValidator(pubKey, validators) {
		return nil, nil
	}

	source := c.lastJustifiedCheckpointOfBranch(target)
	if source != nil {
		v := &Verification{
			SourceHash:   source.Hash,
			TargetHash:   target.Hash,
			SourceHeight: source.Height,
			TargetHeight: target.Height,
			PubKey:       pubKey,
		}

		prvKey := config.CommonConfig.PrivateKey()
		if err := v.Sign(*prvKey); err != nil {
			return nil, err
		}

		if err := c.verifyVerification(v, false); err != nil {
			return nil, nil
		}

		return v, c.addVerificationToCheckpoint(target, v)
	}
	return nil, nil
}

type guarantyArgs struct {
	Amount uint64
	PubKey []byte
}

func isGuarantyProgram(program []byte) bool {
	return false
}

func decodeGuarantyArgs(program []byte) *guarantyArgs {
	return nil
}

func processWithdrawal(guarantyArgs *guarantyArgs, checkpoint *state.Checkpoint) error {
	pubKey := hex.EncodeToString(guarantyArgs.PubKey)
	guarantyNum := checkpoint.Guaranties[pubKey]
	guarantyNum, ok := checked.SubUint64(guarantyNum, guarantyArgs.Amount)
	if !ok {
		return errOverflow
	}

	checkpoint.Guaranties[pubKey] = guarantyNum
	// TODO delete the evil validator when receive the confiscate transaction
	return nil
}

func processGuaranty(guarantyArgs *guarantyArgs, checkpoint *state.Checkpoint) error {
	if guarantyArgs.Amount < minGuaranty {
		return errGuarantyLessThanMinimum
	}

	pubKey := hex.EncodeToString(guarantyArgs.PubKey)
	guarantyNum := checkpoint.Guaranties[pubKey]
	guarantyNum, ok := checked.AddUint64(guarantyNum, guarantyArgs.Amount)
	if !ok {
		return errOverflow
	}

	checkpoint.Guaranties[pubKey] = guarantyNum
	return nil
}

func processVeto(input *types.VetoInput, checkpoint *state.Checkpoint) error {
	pubKey := hex.EncodeToString(input.Vote)
	voteNum := checkpoint.Votes[pubKey]
	voteNum, ok := checked.SubUint64(voteNum, input.Amount)
	if !ok {
		return errOverflow
	}

	checkpoint.Votes[pubKey] = voteNum
	return nil
}

func processVote(output *types.TxOutput, checkpoint *state.Checkpoint) error {
	voteOutput := output.TypedOutput.(*types.VoteOutput)
	pubKey := hex.EncodeToString(voteOutput.Vote)
	if checkpoint.Guaranties[pubKey] < minGuaranty {
		return errVoteToNonValidator
	}

	voteNum := checkpoint.Votes[pubKey]
	voteNum, ok := checked.AddUint64(voteNum, output.Amount)
	if !ok {
		return errOverflow
	}

	checkpoint.Votes[pubKey] = voteNum
	return nil
}

func (c *Casper) lastJustifiedCheckpointOfBranch(branch *state.Checkpoint) *state.Checkpoint {
	parent := branch.Parent
	for parent != nil {
		switch parent.Status {
		case state.Finalized:
			return nil
		case state.Justified:
			return parent
		}
	}
	return nil
}

func supLinkToVerifications(supLink *types.SupLink, validators []*state.Validator, targetHash bc.Hash, targetHeight uint64) []*Verification {
	var result []*Verification
	for i, signature := range supLink.Signatures {
		result = append(result, &Verification{
			SourceHash:   supLink.SourceHash,
			TargetHash:   targetHash,
			SourceHeight: supLink.SourceHeight,
			TargetHeight: targetHeight,
			Signature:    hex.EncodeToString(signature),
			PubKey:       validators[i].PubKey,
		})
	}
	return result
}
