package casper

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

// AuthVerification verify whether the Verification is legal.
// the status of source checkpoint must justified, and an individual validator ν must not publish two distinct Verification
// ⟨ν,s1,t1,h(s1),h(t1)⟩ and ⟨ν,s2,t2,h(s2),h(t2)⟩, such that either:
// h(t1) = h(t2) OR h(s1) < h(s2) < h(t2) < h(t1)
func (c *Casper) AuthVerification(v *Verification) error {
	if err := v.vaild(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// root of tree is the last finalized checkpoint
	if v.TargetHeight < c.tree.Height {
		// discard the verification message which height of target less than height of last finalized checkpoint
		// is for simplify check the vote within the span of its other votes
		return nil
	}

	targetNode, err := c.tree.nodeByHash(v.TargetHash)
	if err != nil {
		c.verificationCache.Add(verificationCacheKey(v.TargetHash, v.PubKey), v)
		return nil
	}

	validators := targetNode.Parent.EffectiveValidators()
	if _, ok := validators[v.PubKey]; !ok {
		return errPubKeyIsNotValidator
	}

	if targetNode.ContainsVerification(validators[v.PubKey].Order, &v.SourceHash) {
		return nil
	}

	oldBestHash := c.bestChain()
	if err := c.authVerification(v, targetNode.Checkpoint, validators); err != nil {
		return err
	}

	return c.tryRollback(oldBestHash)
}

func (c *Casper) authVerification(v *Verification, target *state.Checkpoint, validators map[string]*state.Validator) error {
	validator := validators[v.PubKey]
	if err := c.verifyNested(v, validator.Order); err != nil {
		return err
	}

	checkpoints, err := c.addVerificationToCheckpoint(target, validators, v)
	if err != nil {
		return err
	}

	if err := c.msgQueue.Post(ValidCasperSignEvent{v}); err != nil {
		return err
	}

	if err := c.store.SaveCheckpoints(checkpoints); err != nil {
		return err
	}

	return c.saveVerificationToHeader(v, validator.Order)
}

func (c *Casper) addVerificationToCheckpoint(target *state.Checkpoint, validators map[string]*state.Validator, verifications ...*Verification) ([]*state.Checkpoint, error) {
	affectedCheckpoints := []*state.Checkpoint{target}
	for _, v := range verifications {
		source, err := c.store.GetCheckpoint(&v.SourceHash)
		if err != nil {
			return nil, err
		}

		supLink := target.AddVerification(v.SourceHash, v.SourceHeight, validators[v.PubKey].Order, v.Signature)
		if target.Status != state.Unjustified || !supLink.IsMajority(len(validators)) || source.Status == state.Finalized {
			continue
		}

		c.setJustified(source, target)
		affectedCheckpoints = append(affectedCheckpoints, source)
	}
	return affectedCheckpoints, nil
}

func (c *Casper) saveVerificationToHeader(v *Verification, validatorOrder int) error {
	blockHeader, err := c.store.GetBlockHeader(&v.TargetHash)
	if err != nil {
		return err
	}

	blockHeader.SupLinks.AddSupLink(v.SourceHeight, v.SourceHash, v.Signature, validatorOrder)
	return c.store.SaveBlockHeader(blockHeader)
}

// source status is justified, and exist a super majority link from source to target
func (c *Casper) setJustified(source, target *state.Checkpoint) {
	target.Status = state.Justified
	// must direct child
	if target.ParentHash == source.Hash {
		c.setFinalized(source)
	}
}

func (c *Casper) setFinalized(checkpoint *state.Checkpoint) {
	checkpoint.Status = state.Finalized
	newRoot, err := c.tree.nodeByHash(checkpoint.Hash)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "module": logModule}).Error("source checkpoint before the last finalized checkpoint")
		return
	}

	// update the checkpoint state in memory
	newRoot.Status = state.Finalized
	newRoot.Parent = nil
	c.tree = newRoot
}

func (c *Casper) tryRollback(oldBestHash bc.Hash) error {
	if newBestHash := c.bestChain(); oldBestHash != newBestHash {
		msg := &RollbackMsg{BestHash: newBestHash, Reply: make(chan error)}
		c.rollbackCh <- msg
		return <-msg.Reply
	}
	return nil
}

func (c *Casper) authVerificationLoop() {
	for blockHash := range c.newEpochCh {
		validators, err := c.validators(&blockHash)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "module": logModule}).Error("get validators when auth verification")
			continue
		}

		for _, validator := range validators {
			key := verificationCacheKey(blockHash, validator.PubKey)
			verification, ok := c.verificationCache.Get(key)
			if !ok {
				continue
			}

			v := verification.(*Verification)
			target, err := c.store.GetCheckpoint(&v.TargetHash)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "module": logModule}).Error("get target checkpoint")
				c.verificationCache.Remove(key)
				continue
			}

			c.mu.Lock()
			if err := c.authVerification(v, target, validators); err != nil {
				log.WithFields(log.Fields{"err": err, "module": logModule}).Error("auth verification in cache")
			}
			c.mu.Unlock()

			c.verificationCache.Remove(key)
		}
	}
}

func (c *Casper) verifyNested(v *Verification, validatorOrder int) error {
	if err := c.verifySameHeight(v, validatorOrder); err != nil {
		return err
	}

	return c.verifySpanHeight(v, validatorOrder)
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *Verification, validatorOrder int) error {
	checkpoints, err := c.store.GetCheckpointsByHeight(v.TargetHeight)
	if err != nil {
		return err
	}

	for _, checkpoint := range checkpoints {
		for _, supLink := range checkpoint.SupLinks {
			if len(supLink.Signatures[validatorOrder]) != 0 && checkpoint.Hash != v.TargetHash {
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *Verification, validatorOrder int) error {
	if c.tree.findOnlyOne(func(checkpoint *state.Checkpoint) bool {
		if checkpoint.Height == v.TargetHeight {
			return false
		}

		for _, supLink := range checkpoint.SupLinks {
			if len(supLink.Signatures[validatorOrder]) != 0 {
				if (checkpoint.Height < v.TargetHeight && supLink.SourceHeight > v.SourceHeight) ||
					(checkpoint.Height > v.TargetHeight && supLink.SourceHeight < v.SourceHeight) {
					return true
				}
			}
		}
		return false
	}) != nil {
		return errSpanHeightInVerification
	}
	return nil
}

func verificationCacheKey(blockHash bc.Hash, pubKey string) string {
	return fmt.Sprintf("%s:%s", blockHash.String(), pubKey)
}
