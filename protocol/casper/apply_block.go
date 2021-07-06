package casper

import (
	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

type ApplyBlockReply struct {
	Verification *Verification
	BestHash     bc.Hash
}

// ApplyBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
// it will return verification when an epoch is reached and the current node is the validator, otherwise return nil
// the chain module must broadcast the verification
func (c *Casper) ApplyBlock(block *types.Block) (*ApplyBlockReply, error) {
	if block.Height%consensus.ActiveNetParams.BlocksOfEpoch == 1 {
		c.newEpochCh <- block.PreviousBlockHash
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.tree.nodeByHash(block.Hash()); err == nil {
		return &ApplyBlockReply{BestHash: c.bestChain()}, nil
	}

	target, err := c.applyBlockToCheckpoint(block)
	if err != nil {
		return nil, errors.Wrap(err, "apply block to checkpoint")
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

	return &ApplyBlockReply{Verification: verification, BestHash: c.bestChain()}, c.saveCheckpoints(affectedCheckpoints)
}

func (c *Casper) applyBlockToCheckpoint(block *types.Block) (*state.Checkpoint, error) {
	node, err := c.checkpointNodeByHash(block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	checkpoint := node.Checkpoint
	if mod := block.Height % consensus.ActiveNetParams.BlocksOfEpoch; mod == 1 {
		parent := checkpoint
		checkpoint = state.NewCheckpoint(parent)
		node.addChild(&treeNode{Checkpoint: checkpoint})
	}

	return checkpoint, checkpoint.Increase(block)
}

func (c *Casper) checkpointNodeByHash(blockHash bc.Hash) (*treeNode, error) {
	node, err := c.tree.nodeByHash(blockHash)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err, "module": logModule}).Error("fail find checkpoint, start to reorganize checkpoint")
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

		if prevBlock.Height%consensus.ActiveNetParams.BlocksOfEpoch == 0 {
			break
		}

		attachBlocks = append([]*types.Block{prevBlock}, attachBlocks...)
		prevHash = prevBlock.PreviousBlockHash
	}

	parent, err := c.tree.nodeByHash(prevHash)
	if err != nil {
		return nil, err
	}

	node := &treeNode{Checkpoint: state.NewCheckpoint(parent.Checkpoint)}
	parent.addChild(node)
	for _, attachBlock := range attachBlocks {
		if err := node.Increase(attachBlock); err != nil {
			return nil, err
		}
	}
	return node, nil
}

// applySupLinks copy the block's supLink to the checkpoint
func (c *Casper) applySupLinks(target *state.Checkpoint, supLinks []*types.SupLink, validators map[string]*state.Validator) ([]*state.Checkpoint, error) {
	affectedCheckpoints := []*state.Checkpoint{target}
	if target.Height%consensus.ActiveNetParams.BlocksOfEpoch != 0 {
		return affectedCheckpoints, nil
	}

	for _, supLink := range supLinks {
		var validVerifications []*Verification
		for _, v := range supLinkToVerifications(supLink, validators, target.Hash, target.Height) {
			if validate(v) == nil && c.verifyVerification(v, validators[v.PubKey].Order) == nil {
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

	block.SupLinks.AddSupLink(v.SourceHeight, v.SourceHash, v.Signature, validators[v.PubKey].Order)
	return v, c.store.SaveBlockHeader(&block.BlockHeader)
}

func (c *Casper) myVerification(target *state.Checkpoint, validators map[string]*state.Validator) (*Verification, error) {
	if target.Height%consensus.ActiveNetParams.BlocksOfEpoch != 0 {
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

		if err := c.verifyVerification(v, validatorOrder); err != nil {
			return nil, nil
		}

		return v, nil
	}
	return nil, nil
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
			Signature:    signature,
			PubKey:       validatorList[i].PubKey,
		})
	}
	return result
}
