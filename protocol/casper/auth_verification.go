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
func (c *Casper) AuthVerification(msg *ValidCasperSignMsg) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	targetNode := c.tree.nodeByHash(msg.TargetHash)
	if targetNode == nil {
		c.verificationCache.Add(verificationCacheKey(msg.TargetHash, msg.PubKey), msg)
		return nil
	}

	source, err := c.store.GetCheckpoint(&msg.SourceHash)
	if err != nil {
		return err
	}

	v, err := convertVerification(source, targetNode.Checkpoint, msg)
	if err != nil {
		return err
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

func (c *Casper) authVerification(v *verification, target *state.Checkpoint, validators map[string]*state.Validator) error {
	if err := v.valid(); err != nil {
		return err
	}

	if err := c.verifyNested(v); err != nil {
		return err
	}

	checkpoints, err := c.addVerificationToCheckpoint(target, validators, v)
	if err != nil {
		return err
	}

	if err := c.msgQueue.Post(v.toValidCasperSignMsg()); err != nil {
		return err
	}

	if err := c.store.SaveCheckpoints(checkpoints); err != nil {
		return err
	}

	return c.saveVerificationToHeader(v)
}

func (c *Casper) addVerificationToCheckpoint(target *state.Checkpoint, validators map[string]*state.Validator, verifications ...*verification) ([]*state.Checkpoint, error) {
	affectedCheckpoints := []*state.Checkpoint{target}
	for _, v := range verifications {
		source, err := c.store.GetCheckpoint(&v.SourceHash)
		if err != nil {
			return nil, err
		}

		supLink := target.AddVerification(v.SourceHash, v.SourceHeight, v.order, v.Signature)
		if target.Status != state.Unjustified || !supLink.IsMajority(len(validators)) || source.Status == state.Finalized {
			continue
		}

		c.setJustified(source, target)
		affectedCheckpoints = append(affectedCheckpoints, source)
	}
	return affectedCheckpoints, nil
}

func (c *Casper) saveVerificationToHeader(v *verification) error {
	blockHeader, err := c.store.GetBlockHeader(&v.TargetHash)
	if err != nil {
		return err
	}

	blockHeader.SupLinks.AddSupLink(v.SourceHeight, v.SourceHash, v.Signature, v.order)
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
	newRoot := c.tree.nodeByHash(checkpoint.Hash)
	if newRoot == nil {
		log.WithFields(log.Fields{"module": logModule, "hash": checkpoint.Hash}).Warn("source checkpoint before the last finalized checkpoint")
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
			data, ok := c.verificationCache.Get(key)
			if !ok {
				continue
			}

			msg := data.(*ValidCasperSignMsg)
			source, err := c.store.GetCheckpoint(&msg.SourceHash)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "module": logModule}).Error("get source checkpoint")
				c.verificationCache.Remove(key)
			}

			target, err := c.store.GetCheckpoint(&msg.TargetHash)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "module": logModule}).Error("get target checkpoint")
				c.verificationCache.Remove(key)
				continue
			}

			v, err := convertVerification(source, target, msg)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "module": logModule}).Error("authVerificationLoop fail on newVerification")
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

func (c *Casper) verifyNested(v *verification) error {
	if err := c.verifySameHeight(v); err != nil {
		return err
	}

	return c.verifySpanHeight(v)
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *verification) error {
	checkpoints, err := c.store.GetCheckpointsByHeight(v.TargetHeight)
	if err != nil {
		return err
	}

	for _, checkpoint := range checkpoints {
		for _, supLink := range checkpoint.SupLinks {
			if len(supLink.Signatures[v.order]) != 0 && checkpoint.Hash != v.TargetHash {
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *verification) error {
	if c.tree.findOnlyOne(func(checkpoint *state.Checkpoint) bool {
		if checkpoint.Height == v.TargetHeight {
			return false
		}

		for _, supLink := range checkpoint.SupLinks {
			if len(supLink.Signatures[v.order]) != 0 {
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
