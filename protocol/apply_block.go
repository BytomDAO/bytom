package protocol

import (
	"encoding/hex"

	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/math/checked"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

type applyBlockReply struct {
	verification *Verification
	bestHash     bc.Hash
}

// ApplyBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
// it will return verification when an epoch is reached and the current node is the validator, otherwise return nil
// the chain module must broadcast the verification
func (c *Casper) ApplyBlock(block *types.Block) (*applyBlockReply, error) {
	if block.Height%state.BlocksOfEpoch == 1 {
		c.newEpochCh <- block.PreviousBlockHash
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.tree.nodeByHash(block.Hash()); err == nil {
		return &applyBlockReply{bestHash: c.bestChain()}, nil
	}

	target, err := c.applyBlockToCheckpoint(block)
	if err != nil {
		return nil, errors.Wrap(err, "apply block to checkpoint")
	}

	if err := applyTransactions(target, block.Transactions); err != nil {
		return nil, err
	}

	validators, err := c.Validators(&target.Hash)
	if err != nil {
		return nil, err
	}

	verification, err := c.applyMyVerification(target, block, validators)
	if err != nil {
		return nil, err
	}

	affectedCheckpoints, err := c.applySupLinks(target, block.SupLinks, validators)
	if err != nil {
		return nil, err
	}

	return &applyBlockReply{verification: verification, bestHash: c.bestChain()}, c.saveCheckpoints(affectedCheckpoints)
}

func (c *Casper) applyBlockToCheckpoint(block *types.Block) (*state.Checkpoint, error) {
	node, err := c.checkpointNodeByHash(block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	checkpoint := node.checkpoint
	if mod := block.Height % state.BlocksOfEpoch; mod == 1 {
		parent := checkpoint
		checkpoint = &state.Checkpoint{
			ParentHash: parent.Hash,
			Parent:     parent,
			Status:     state.Growing,
			Votes:      make(map[string]uint64),
			Guaranties: make(map[string]uint64),
		}

		for pubKey, num := range parent.Votes {
			checkpoint.Votes[pubKey] = num
		}
		for pubKey, num := range parent.Guaranties {
			checkpoint.Guaranties[pubKey] = num
		}
		node.addChild(&treeNode{checkpoint: checkpoint})
	} else if mod == 0 {
		checkpoint.Status = state.Unjustified
	}

	checkpoint.Height = block.Height
	checkpoint.Hash = block.Hash()
	checkpoint.Timestamp = block.Timestamp
	return checkpoint, nil
}

func (c *Casper) checkpointNodeByHash(blockHash bc.Hash) (*treeNode, error) {
	node, err := c.tree.nodeByHash(blockHash)
	if err != nil {
		logrus.WithField("err", err).Error("fail find checkpoint, start to reorganize checkpoint")

		return c.replayCheckpoint(blockHash)
	}

	return node, nil
}

func (c *Casper) replayCheckpoint(hash bc.Hash) (*treeNode, error) {
	prevHash := hash
	var attachBlocks []*types.Block
	for {
		prevBlock, err := c.store.GetBlock(&prevHash)
		if err != nil {
			return nil, err
		}

		if prevBlock.Height%state.BlocksOfEpoch == 0 {
			break
		}

		attachBlocks = append([]*types.Block{prevBlock}, attachBlocks...)
		prevHash = prevBlock.PreviousBlockHash
	}

	parent, err := c.tree.nodeByHash(prevHash)
	if err != nil {
		return nil, err
	}

	node := &treeNode{
		checkpoint: &state.Checkpoint{
			ParentHash: parent.checkpoint.Hash,
			Parent:     parent.checkpoint,
			Status:     state.Growing,
			Votes:      make(map[string]uint64),
			Guaranties: make(map[string]uint64),
		},
	}

	parent.addChild(node)
	for _, attachBlock := range attachBlocks {
		if err := applyTransactions(node.checkpoint, attachBlock.Transactions); err != nil {
			return nil, err
		}

		node.checkpoint.Hash = attachBlock.Hash()
		node.checkpoint.Height = attachBlock.Height
		node.checkpoint.Timestamp = attachBlock.Timestamp
	}
	return node, nil
}

func applyTransactions(target *state.Checkpoint, transactions []*types.Tx) error {
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
func (c *Casper) applySupLinks(target *state.Checkpoint, supLinks []*types.SupLink, validators map[string]*state.Validator) ([]*state.Checkpoint, error) {
	affectedCheckpoints := []*state.Checkpoint{target}
	if target.Height%state.BlocksOfEpoch != 0 {
		return nil, nil
	}

	for _, supLink := range supLinks {
		var validVerifications []*Verification
		for _, v := range supLinkToVerifications(supLink, validators, target.Hash, target.Height) {
			if validate(v) == nil && c.verifyVerification(v, validators[v.PubKey].Order, true) == nil {
				validVerifications = append(validVerifications, v)
			}
		}

		checkpoints, err := c.addVerificationToCheckpoint(target, validators, validVerifications...)
		if err != nil {
			return nil, err
		}

		affectedCheckpoints = append(affectedCheckpoints, checkpoints...)
	}
	return affectedCheckpoints, nil
}

func (c *Casper) applyMyVerification(target *state.Checkpoint, block *types.Block, validators map[string]*state.Validator) (*Verification, error) {
	v, err := c.myVerification(target, validators)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	signature, err := hex.DecodeString(v.Signature)
	if err != nil {
		return nil, err
	}

	block.SupLinks.AddSupLink(v.SourceHeight, v.SourceHash, signature, validators[v.PubKey].Order)
	return v, c.store.SaveBlockHeader(&block.BlockHeader)
}

func (c *Casper) myVerification(target *state.Checkpoint, validators map[string]*state.Validator) (*Verification, error) {
	if target.Height%state.BlocksOfEpoch != 0 {
		return nil, nil
	}

	pubKey := config.CommonConfig.PrivateKey().XPub().String()
	if _, ok := validators[pubKey]; !ok {
		return nil, nil
	}

	validatorOrder := validators[pubKey].Order
	if target.ContainsVerification(validatorOrder, nil) {
		return nil, nil
	}

	if source := c.lastJustifiedCheckpoint(target); source != nil {
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

		if err := c.verifyVerification(v, validatorOrder, false); err != nil {
			return nil, nil
		}

		return v, nil
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

func (c *Casper) lastJustifiedCheckpoint(branch *state.Checkpoint) *state.Checkpoint {
	parent := branch.Parent
	for parent != nil {
		switch parent.Status {
		case state.Finalized:
			return nil
		case state.Justified:
			return parent
		}
		parent = parent.Parent
	}
	return nil
}

func (c *Casper) saveCheckpoints(checkpoints []*state.Checkpoint) error {
	checkpointSet := make(map[bc.Hash]*state.Checkpoint)
	for _, c := range checkpoints {
		checkpointSet[c.Hash] = c
	}

	var result []*state.Checkpoint
	for _, c := range checkpointSet {
		result = append(result, c)
	}

	return c.store.SaveCheckpoints(result)
}

func supLinkToVerifications(supLink *types.SupLink, validators map[string]*state.Validator, targetHash bc.Hash, targetHeight uint64) []*Verification {
	validatorList := make([]*state.Validator, len(validators))
	for _, validator := range validators {
		validatorList[validator.Order] = validator
	}

	var result []*Verification
	for i := 0; i < len(validators); i++ {
		signature := supLink.Signatures[i]
		if len(signature) == 0 {
			continue
		}

		result = append(result, &Verification{
			SourceHash:   supLink.SourceHash,
			TargetHash:   targetHash,
			SourceHeight: supLink.SourceHeight,
			TargetHeight: targetHeight,
			Signature:    hex.EncodeToString(signature),
			PubKey:       validatorList[i].PubKey,
		})
	}
	return result
}
