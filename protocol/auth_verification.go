package protocol

import (
	"encoding/hex"
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

	target, err := c.store.GetCheckpoint(&v.TargetHash)
	if err != nil {
		c.verificationCache.Add(verificationCacheKey(v.TargetHash, v.PubKey), v)
		return nil
	}

	validators, err := c.Validators(&v.TargetHash)
	if err != nil {
		return err
	}

	 if _, ok := validators[v.PubKey]; !ok {
		return errPubKeyIsNotValidator
	}

	if target.ContainsVerification(validators[v.PubKey].Order, &v.SourceHash) {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// root of tree is the last finalized checkpoint
	if v.TargetHeight < c.tree.checkpoint.Height {
		// discard the verification message which height of target less than height of last finalized checkpoint
		// is for simplify check the vote within the span of its other votes
		return nil
	}

	return c.authVerification(v, target, validators)
}

func (c *Casper) authVerification(v *Verification, target *state.Checkpoint, validators map[string]*state.Validator) error {
	validator := validators[v.PubKey]
	if err := c.verifyVerification(v, validator.Order, true); err != nil {
		return err
	}

	checkpoints, err := c.addVerificationToCheckpoint(target, validators, v)
	if err != nil {
		return err
	}

	if err := c.store.SaveCheckpoints(checkpoints); err != nil {
		return err
	}

	return c.saveVerificationToHeader(v, validator.Order)
}

func (c *Casper) addVerificationToCheckpoint(target *state.Checkpoint, validators map[string]*state.Validator, verifications ...*Verification) ([]*state.Checkpoint, error) {
	_, oldBestHash := c.bestChain()
	var affectedCheckpoints []*state.Checkpoint
	for _, v := range verifications {
		source, err := c.store.GetCheckpoint(&v.SourceHash)
		if err != nil {
			return nil, err
		}

		supLink := target.AddVerification(v.SourceHash, v.SourceHeight, validators[v.PubKey].Order, v.Signature)
		if target.Status != state.Unjustified || !supLink.IsMajority(len(validators)) || source.Status == state.Finalized {
			continue
		}

		if source.Status == state.Unjustified {
			c.justifyingCheckpoints[source.Hash] = append(c.justifyingCheckpoints[source.Hash], target)
			continue
		}

		affectedCheckpoints = append(affectedCheckpoints, c.setJustified(source, target)...)
	}

	_, newBestHash := c.bestChain()
	if oldBestHash != newBestHash {
		c.rollbackNotifyCh <- newBestHash
	}

	return affectedCheckpoints, nil
}

func (c *Casper) saveVerificationToHeader(v *Verification, validatorOrder int) error {
	blockHeader, err := c.store.GetBlockHeader(&v.TargetHash)
	if err != nil {
		return err
	}

	signature, err := hex.DecodeString(v.Signature)
	if err != nil {
		return err
	}

	blockHeader.SupLinks.AddSupLink(v.SourceHeight, v.SourceHash, signature, validatorOrder)
	return c.store.SaveBlockHeader(blockHeader)
}

// source status is justified, and exist a super majority link from source to target
func (c *Casper) setJustified(source, target *state.Checkpoint) []*state.Checkpoint {
	var affectedCheckpoint []*state.Checkpoint
	target.Status = state.Justified
	// must direct child
	if target.Parent.Hash == source.Hash {
		c.setFinalized(source)
	}

	for _, checkpoint := range c.justifyingCheckpoints[target.Hash] {
		affectedCheckpoint = append(affectedCheckpoint, c.setJustified(target, checkpoint)...)
	}

	delete(c.justifyingCheckpoints, target.Hash)
	affectedCheckpoint = append(affectedCheckpoint, source, target)
	return affectedCheckpoint
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

			v := verification.(*Verification)
			target, err := c.store.GetCheckpoint(&v.TargetHash)
			if err != nil {
				log.WithField("err", err).Error("get target checkpoint")
				c.verificationCache.Remove(key)
				continue
			}

			c.mu.Lock()
			if err := c.authVerification(v, target, validators); err != nil {
				log.WithField("err", err).Error("auth verification in cache")
			}
			c.mu.Unlock()

			c.verificationCache.Remove(key)
		}
	}
}

func (c *Casper) verifyVerification(v *Verification, validatorOrder int, trackEvilValidator bool) error {
	if err := c.verifySameHeight(v, validatorOrder, trackEvilValidator); err != nil {
		return err
	}

	return c.verifySpanHeight(v, validatorOrder, trackEvilValidator)
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *Verification, validatorOrder int, trackEvilValidator bool) error {
	checkpoints, err := c.store.GetCheckpointsByHeight(v.TargetHeight)
	if err != nil {
		return err
	}

	for _, checkpoint := range checkpoints {
		for _, supLink := range checkpoint.SupLinks {
			if supLink.Signatures[validatorOrder] != "" && checkpoint.Hash != v.TargetHash {
				if trackEvilValidator {
					c.evilValidators[v.PubKey] = []*Verification{v, makeVerification(supLink, checkpoint, v.PubKey, validatorOrder)}
				}
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *Verification, validatorOrder int, trackEvilValidator bool) error {
	if c.tree.findOnlyOne(func(checkpoint *state.Checkpoint) bool {
		if checkpoint.Height == v.TargetHeight {
			return false
		}

		for _, supLink := range checkpoint.SupLinks {
			if supLink.Signatures[validatorOrder] != "" {
				if (checkpoint.Height < v.TargetHeight && supLink.SourceHeight > v.SourceHeight) ||
					(checkpoint.Height > v.TargetHeight && supLink.SourceHeight < v.SourceHeight) {
					if trackEvilValidator {
						c.evilValidators[v.PubKey] = []*Verification{v, makeVerification(supLink, checkpoint, v.PubKey, validatorOrder)}
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

func makeVerification(supLink *state.SupLink, checkpoint *state.Checkpoint, pubKey string, validatorOrder int) *Verification {
	return &Verification{
		SourceHash:   supLink.SourceHash,
		TargetHash:   checkpoint.Hash,
		SourceHeight: supLink.SourceHeight,
		TargetHeight: checkpoint.Height,
		Signature:    supLink.Signatures[validatorOrder],
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
