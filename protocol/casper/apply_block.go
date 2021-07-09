package casper

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

// ApplyBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
// it will return verification when an epoch is reached and the current node is the validator, otherwise return nil
// the chain module must broadcast the verification
func (c *Casper) ApplyBlock(block *types.Block) (bc.Hash, error) {
	if block.Height%consensus.ActiveNetParams.BlocksOfEpoch == 1 {
		c.newEpochCh <- block.PreviousBlockHash
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if node := c.tree.nodeByHash(block.Hash()); node != nil {
		return c.bestChain(), nil
	}

	target, err := c.applyBlockToCheckpoint(block)
	if err != nil {
		return bc.Hash{}, errors.Wrap(err, "apply block to checkpoint")
	}

	if err := c.applyMyVerification(target, block); err != nil {
		return bc.Hash{}, err
	}

	affectedCheckpoints, err := c.applySupLinks(target, block.SupLinks)
	if err != nil {
		return bc.Hash{}, err
	}

	return c.bestChain(), c.saveCheckpoints(affectedCheckpoints)
}

func (c *Casper) applyBlockToCheckpoint(block *types.Block) (*state.Checkpoint, error) {
	node, err := c.checkpointNodeByHash(block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	if block.Height%consensus.ActiveNetParams.BlocksOfEpoch == 1 {
		node = node.newChild()
	}
	return node.Checkpoint, node.Increase(block)
}

func (c *Casper) checkpointNodeByHash(hash bc.Hash) (*treeNode, error) {
	if node := c.tree.nodeByHash(hash); node != nil {
		return node, nil
	}

	block, err := c.store.GetBlock(&hash)
	if err != nil {
		return nil, err
	}

	if block.Height%consensus.ActiveNetParams.BlocksOfEpoch == 0 {
		return nil, errors.New("checkpointNodeByHash fail on previous round checkpoint")
	}

	node, err := c.checkpointNodeByHash(block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	if block.Height%consensus.ActiveNetParams.BlocksOfEpoch == 1 {
		node = node.newChild()
	}
	return node, node.Increase(block)
}

// applySupLinks copy the block's supLink to the checkpoint
func (c *Casper) applySupLinks(target *state.Checkpoint, supLinks []*types.SupLink) ([]*state.Checkpoint, error) {
	validators := target.Parent.EffectiveValidators()
	affectedCheckpoints := []*state.Checkpoint{target}
	if target.Height%consensus.ActiveNetParams.BlocksOfEpoch != 0 {
		return affectedCheckpoints, nil
	}

	for _, supLink := range supLinks {
		var validVerifications []*verification
		for _, v := range supLinkToVerifications(supLink, validators, target.Hash, target.Height) {
			if v.valid() == nil && c.verifyNested(v) == nil {
				validVerifications = append(validVerifications, v)
			}
		}

		checkpoints, err := c.addVerificationToCheckpoint(target, validVerifications...)
		if err != nil {
			return nil, err
		}

		affectedCheckpoints = append(affectedCheckpoints, checkpoints...)
	}
	return affectedCheckpoints, nil
}

func (c *Casper) applyMyVerification(target *state.Checkpoint, block *types.Block) error {
	v := c.myVerification(target)
	if v == nil {
		return nil
	}

	if err := c.msgQueue.Post(v.toValidCasperSignMsg()); err != nil {
		return err
	}

	block.SupLinks.AddSupLink(v.SourceHeight, v.SourceHash, v.Signature, v.order)
	return c.store.SaveBlockHeader(&block.BlockHeader)
}

func (c *Casper) myVerification(target *state.Checkpoint) *verification {
	if target.Status == state.Growing {
		return nil
	}

	source := lastJustifiedCheckpoint(target)
	if source == nil {
		log.WithField("module", logModule).Warn("myVerification fail on find last justified")
		return nil
	}

	prvKey := config.CommonConfig.PrivateKey()
	v, err := convertVerification(source, target, &ValidCasperSignMsg{PubKey: prvKey.XPub().String()})
	if err != nil {
		return nil
	}

	if target.ContainsVerification(v.order, nil) {
		log.WithField("module", logModule).Warn("myVerification fail on find same height sign")
		return nil
	}

	if err := c.verifyNested(v); err != nil {
		log.WithField("module", logModule).Warn("myVerification fail on find nest sign")
		return nil
	}

	if err := v.Sign(*prvKey); err != nil {
		log.WithField("module", logModule).Error("myVerification fail on sign msg")
		return nil
	}

	return v
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

func lastJustifiedCheckpoint(branch *state.Checkpoint) *state.Checkpoint {
	for parent := branch.Parent; parent != nil; parent = parent.Parent {
		if parent.Status == state.Justified {
			return parent
		}
	}
	return nil
}

func supLinkToVerifications(supLink *types.SupLink, validators map[string]*state.Validator, targetHash bc.Hash, targetHeight uint64) []*verification {
	validatorList := make([]*state.Validator, len(validators))
	for _, validator := range validators {
		validatorList[validator.Order] = validator
	}

	var result []*verification
	for i := 0; i < len(validators); i++ {
		signature := supLink.Signatures[i]
		if len(signature) == 0 {
			continue
		}

		result = append(result, &verification{
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
