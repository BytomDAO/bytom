package protocol

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
	if err := validate(v); err != nil {
		return err
	}

	validators, err := c.Validators(&v.TargetHash)
	if err != nil {
		return err
	}

	if !isValidator(v.PubKey, validators) {
		return errPubKeyIsNotValidator
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// root of tree is the last finalized checkpoint
	if v.TargetHeight < c.tree.checkpoint.Height {
		// discard the verification message which height of target less than height of last finalized checkpoint
		// is for simplify check the vote within the span of its other votes
		return nil
	}

	return c.authVerification(v)
}

func (c *Casper) authVerification(v *Verification) error {
	target, err := c.store.GetCheckpoint(&v.TargetHash)
	if err != nil {
		c.verificationCache.Add(verificationCacheKey(v.TargetHash, v.PubKey), v)
		return nil
	}

	if err := c.verifyVerification(v, true); err != nil {
		return err
	}

	return c.addVerificationToCheckpoint(target, v)
}

func (c *Casper) addVerificationToCheckpoint(target *state.Checkpoint, v *Verification) error {
	source, err := c.store.GetCheckpoint(&v.SourceHash)
	if err != nil {
		return err
	}

	supLink := target.AddVerification(v.SourceHash, v.SourceHeight, v.PubKey, v.Signature)
	if target.Status != state.Unjustified || !supLink.IsMajority() || source.Status == state.Finalized {
		return nil
	}

	if source.Status == state.Unjustified {
		c.justifyingCheckpoints[source.Hash] = append(c.justifyingCheckpoints[source.Hash], target)
		return nil
	}

	_, oldBestHash := c.BestChain()
	affectedCheckpoints := c.setJustified(source, target)
	_, newBestHash := c.BestChain()
	if oldBestHash != newBestHash {
		c.rollbackNotifyCh <- nil
	}

	return c.store.SaveCheckpoints(affectedCheckpoints...)
}

func (c *Casper) setJustified(source, target *state.Checkpoint) []*state.Checkpoint {
	affectedCheckpoints := make(map[bc.Hash]*state.Checkpoint)
	target.Status = state.Justified
	affectedCheckpoints[target.Hash] = target
	// must direct child
	if target.Parent.Hash == source.Hash {
		c.setFinalized(source)
		affectedCheckpoints[source.Hash] = source
	}

	for _, checkpoint := range c.justifyingCheckpoints[target.Hash] {
		for _, c := range c.setJustified(target, checkpoint) {
			affectedCheckpoints[c.Hash] = c
		}
	}
	delete(c.justifyingCheckpoints, target.Hash)

	var result []*state.Checkpoint
	for _, c := range affectedCheckpoints {
		result = append(result, c)
	}
	return result
}

func (c *Casper) setFinalized(checkpoint *state.Checkpoint) {
	checkpoint.Status = state.Finalized
	newRoot, err := c.tree.nodeByHash(checkpoint.Hash)
	if err != nil {
		log.WithField("err", err).Panic("fail to set checkpoint finalized")
	}

	c.tree = newRoot
}

func (c *Casper) authVerificationLoop() {
	for blockHash := range c.newEpochCh {
		validators, err := c.Validators(&blockHash)
		if err != nil {
			log.WithField("err", err).Error("get validators when auth verification")
			continue
		}

		for _, validator := range validators {
			key := verificationCacheKey(blockHash, validator.PubKey)
			verification, ok := c.verificationCache.Get(key)
			if !ok {
				continue
			}

			c.mu.Lock()
			if err := c.authVerification(verification.(*Verification)); err != nil {
				log.WithField("err", err).Error("auth verification in cache")
			}
			c.mu.Unlock()

			c.verificationCache.Remove(key)
		}
	}
}

func (c *Casper) verifyVerification(v *Verification, trackEvilValidator bool) error {
	if err := c.verifySameHeight(v, trackEvilValidator); err != nil {
		return err
	}

	return c.verifySpanHeight(v, trackEvilValidator)
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *Verification, trackEvilValidator bool) error {
	checkpoints, err := c.store.GetCheckpointsByHeight(v.TargetHeight)
	if err != nil {
		return err
	}

	for _, checkpoint := range checkpoints {
		for _, supLink := range checkpoint.SupLinks {
			if _, ok := supLink.Signatures[v.PubKey]; ok && checkpoint.Hash != v.TargetHash {
				if trackEvilValidator {
					c.evilValidators[v.PubKey] = []*Verification{v, makeVerification(supLink, checkpoint, v.PubKey)}
				}
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *Verification, trackEvilValidator bool) error {
	if c.tree.findOnlyOne(func(checkpoint *state.Checkpoint) bool {
		if checkpoint.Height == v.TargetHeight {
			return false
		}

		for _, supLink := range checkpoint.SupLinks {
			if _, ok := supLink.Signatures[v.PubKey]; ok {
				if (checkpoint.Height < v.TargetHeight && supLink.SourceHeight > v.SourceHeight) ||
					(checkpoint.Height > v.TargetHeight && supLink.SourceHeight < v.SourceHeight) {
					if trackEvilValidator {
						c.evilValidators[v.PubKey] = []*Verification{v, makeVerification(supLink, checkpoint, v.PubKey)}
					}
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

func makeVerification(supLink *state.SupLink, checkpoint *state.Checkpoint, pubKey string) *Verification {
	return &Verification{
		SourceHash:   supLink.SourceHash,
		TargetHash:   checkpoint.Hash,
		SourceHeight: supLink.SourceHeight,
		TargetHeight: checkpoint.Height,
		Signature:    supLink.Signatures[pubKey],
		PubKey:       pubKey,
	}
}

func validate(v *Verification) error {
	if v.SourceHeight%state.BlocksOfEpoch != 0 || v.TargetHeight%state.BlocksOfEpoch != 0 {
		return errVoteToGrowingCheckpoint
	}

	if v.SourceHeight == v.TargetHeight {
		return errVoteToSameCheckpoint
	}

	return v.VerifySignature()
}

func verificationCacheKey(blockHash bc.Hash, pubKey string) string {
	return fmt.Sprintf("%s:%s", blockHash.String(), pubKey)
}
