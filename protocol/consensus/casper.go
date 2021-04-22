package consensus

import (
	"encoding/hex"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/math/checked"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

var (
	errVerifySignature          = errors.New("signature of verification message is invalid")
	errPubKeyIsNotValidator     = errors.New("pub key is not in validators of target checkpoint")
	errVoteToGrowingCheckpoint  = errors.New("validator publish vote to growing checkpoint")
	errVoteToSameCheckpoint     = errors.New("source height and target height in verification is equals")
	errSameHeightInVerification = errors.New("validator publish two distinct votes for the same target height")
	errSpanHeightInVerification = errors.New("validator publish vote within the span of its other votes")
	errVoteToNonValidator       = errors.New("pubKey of vote is not validator")
	errGuarantyLessThanMinimum  = errors.New("guaranty less than minimum")
	errOverflow                 = errors.New("arithmetic overflow/underflow")
)

const minGuaranty = 1E14

// Casper is BFT based proof of stack consensus algorithm, it provides safety and liveness in theory,
// it's design mainly refers to https://github.com/ethereum/research/blob/master/papers/casper-basics/casper_basics.pdf
type Casper struct {
	mu               sync.RWMutex
	tree             *treeNode
	rollbackNotifyCh chan bc.Hash
	newEpochCh       chan bc.Hash
	store            protocol.Store
	// pubKey -> conflicting verifications
	evilValidators map[string][]*Verification
	// block hash -> previous checkpoint hash
	prevCheckpointCache *common.Cache
	// block hash + pubKey -> verification
	verificationCache *common.Cache
}

// Best chain return the chain containing the justified checkpoint of the largest height
func (c *Casper) BestChain() (uint64, bc.Hash) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// root is init justified
	root := c.tree.checkpoint
	bestHeight, bestHash, _ := chainOfMaxJustifiedHeight(c.tree, root.Height)
	return bestHeight, bestHash
}

// Validators return the validators by specified block hash
// e.g. if the block num of epoch is 100, and the block height corresponding to the block hash is 130, then will return the voting results of height in 0~100
func (c *Casper) Validators(blockHash *bc.Hash) ([]*state.Validator, error) {
	checkpoint, err := c.prevCheckpoint(blockHash)
	if err != nil {
		return nil, err
	}

	return checkpoint.Validators(), nil
}

// AuthVerification verify whether the Verification is legal.
// the status of source checkpoint must justified, and an individual validator ν must not publish two distinct Verification
// ⟨ν,s1,t1,h(s1),h(t1)⟩ and ⟨ν,s2,t2,h(s2),h(t2)⟩, such that either:
// h(t1) = h(t2) OR h(s1) < h(s2) < h(t2) < h(t1)
func (c *Casper) AuthVerification(v *Verification) error {
	if err := v.validate(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// root of tree is the last finalized checkpoint
	if v.TargetHeight < c.tree.checkpoint.Height {
		// discard the verification message which height of target less than height of last finalized checkpoint
		// is for simplify check the vote within the span of its other votes
		return nil
	}

	prevCheckpoint, err := c.prevCheckpoint(&v.TargetHash)
	if err != nil {
		return err
	}

	if !prevCheckpoint.ContainsValidator(v.PubKey) {
		return errPubKeyIsNotValidator
	}

	return c.authVerification(v)
}

func (c *Casper) authVerification(v *Verification) error {
	target, err := c.store.GetCheckpoint(&v.TargetHash)
	if err != nil {
		c.verificationCache.Add(verificationCacheKey(v.TargetHash, v.PubKey), v)
		return nil
	}

	source, err := c.store.GetCheckpoint(&v.SourceHash)
	if err != nil {
		return err
	}

	if err := c.verifyVerification(v); err != nil {
		return err
	}

	supLink := target.AddSupLink(v.SourceHeight, v.SourceHash, v.PubKey, v.Signature)
	if source.Status == state.Justified && supLink.Confirmed() {
		c.setJustified(target)
		// must direct child
		if target.PrevHash == source.Hash {
			if err := c.setFinalized(source); err != nil {
				return err
			}
		}
	}
	return c.store.SaveCheckpoints(source, target)
}

func verificationCacheKey(blockHash bc.Hash, pubKey string) string {
	return fmt.Sprintf("%s:%s", blockHash.String(), pubKey)
}

func (c *Casper) setJustified(checkpoint *state.Checkpoint) {
	_, oldBestHash := c.BestChain()
	checkpoint.Status = state.Justified
	if _, bestHash := c.BestChain(); bestHash != oldBestHash {
		c.rollbackNotifyCh <- bestHash
	}
}

func (c *Casper) setFinalized(checkpoint *state.Checkpoint) error {
	checkpoint.Status = state.Finalized
	newRoot, err := c.tree.nodeByHash(checkpoint.Hash)
	if err != nil {
		return err
	}

	c.tree = newRoot
	return nil
}

// EvilValidator represent a validator who broadcast two distinct verification that violate the commandment
type EvilValidator struct {
	PubKey string
	V1     *Verification
	V2     *Verification
}

// EvilValidators return all evil validators
func (c *Casper) EvilValidators() []*EvilValidator {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var validators []*EvilValidator
	for pubKey, verifications := range c.evilValidators {
		validators = append(validators, &EvilValidator{
			PubKey: pubKey,
			V1:     verifications[0],
			V2:     verifications[1],
		})
	}
	return validators
}

// ApplyBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
func (c *Casper) ApplyBlock(block *types.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.tree.nodeByHash(block.Hash()); err == nil {
		// already processed
		return nil
	}

	checkpoint, err := c.applyBlockToCheckpoint(block)
	if err != nil {
		return errors.Wrap(err, "apply block to checkpoint")
	}

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if vetoInput, ok := input.TypedInput.(*types.VetoInput); ok {
				if err := processVeto(vetoInput, checkpoint); err != nil {
					return err
				}
			}

			if isGuarantyProgram(input.ControlProgram()) {
				if err := processWithdrawal(decodeGuarantyArgs(input.ControlProgram()), checkpoint); err != nil {
					return err
				}
			}
		}

		for _, output := range tx.Outputs {
			if _, ok := output.TypedOutput.(*types.VoteOutput); ok {
				if err := processVote(output, checkpoint); err != nil {
					return err
				}
			}

			if isGuarantyProgram(output.ControlProgram) {
				if err := processGuaranty(decodeGuarantyArgs(output.ControlProgram), checkpoint); err != nil {
					return err
				}
			}
		}
	}

	if err := c.store.SaveCheckpoints(checkpoint); err != nil {
		return err
	}

	if block.Height%state.BlocksOfEpoch == 0 {
		c.newEpochCh <- block.Hash()
	}
	return nil
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

func (c *Casper) applyBlockToCheckpoint(block *types.Block) (*state.Checkpoint, error) {
	node, err := c.tree.nodeByHash(block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	checkpoint := node.checkpoint
	if mod := block.Height % state.BlocksOfEpoch; mod == 1 {
		parent := checkpoint
		checkpoint = &state.Checkpoint{
			PrevHash:       parent.Hash,
			StartTimestamp: block.Timestamp,
			Status:         state.Growing,
			Votes:          make(map[string]uint64),
			Guaranties:     make(map[string]uint64),
		}
		node.children = append(node.children, &treeNode{checkpoint: checkpoint})
	} else if mod == 0 {
		checkpoint.Status = state.Unverified
	}

	checkpoint.Height = block.Height
	checkpoint.Hash = block.Hash()
	return checkpoint, nil
}

func (c *Casper) verifyVerification(v *Verification) error {
	if err := c.verifySameHeight(v); err != nil {
		return err
	}

	return c.verifySpanHeight(v)
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *Verification) error {
	checkpoints, err := c.store.GetCheckpointsByHeight(v.TargetHeight)
	if err != nil {
		return err
	}

	for _, checkpoint := range checkpoints {
		for _, supLink := range checkpoint.SupLinks {
			if _, ok := supLink.Signatures[v.PubKey]; ok {
				c.evilValidators[v.PubKey] = []*Verification{v, makeVerification(supLink, checkpoint, v.PubKey)}
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *Verification) error {
	if c.tree.findOnlyOne(func(checkpoint *state.Checkpoint) bool {
		if checkpoint.Height == v.TargetHeight {
			return false
		}

		for _, supLink := range checkpoint.SupLinks {
			if _, ok := supLink.Signatures[v.PubKey]; ok {
				if (checkpoint.Height < v.TargetHeight && supLink.SourceHeight > v.SourceHeight) ||
					(checkpoint.Height > v.TargetHeight && supLink.SourceHeight < v.SourceHeight) {
					c.evilValidators[v.PubKey] = []*Verification{v, makeVerification(supLink, checkpoint, v.PubKey)}
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

// justifiedHeight is the max justified height of checkpoint from node to root
func chainOfMaxJustifiedHeight(node *treeNode, justifiedHeight uint64) (uint64, bc.Hash, uint64) {
	checkpoint := node.checkpoint
	if checkpoint.Status == state.Justified {
		justifiedHeight = checkpoint.Height
	}

	bestHeight, bestHash, maxJustifiedHeight := checkpoint.Height, checkpoint.Hash, justifiedHeight
	for _, child := range node.children {
		if height, hash, justified := chainOfMaxJustifiedHeight(child, justifiedHeight); justified >= maxJustifiedHeight {
			bestHeight, bestHash, maxJustifiedHeight = height, hash, justified
		}
	}
	return bestHeight, bestHash, maxJustifiedHeight
}

func (c *Casper) prevCheckpoint(blockHash *bc.Hash) (*state.Checkpoint, error) {
	hash, err := c.prevCheckpointHash(blockHash)
	if err != nil {
		return nil, err
	}

	return c.store.GetCheckpoint(hash)
}

func (c *Casper) prevCheckpointHash(blockHash *bc.Hash) (*bc.Hash, error) {
	if data, ok := c.prevCheckpointCache.Get(*blockHash); ok {
		return data.(*bc.Hash), nil
	}

	for {
		block, err := c.store.GetBlockHeader(blockHash)
		if err != nil {
			return nil, err
		}

		prevHeight, prevHash := block.Height-1, block.PreviousBlockHash
		if data, ok := c.prevCheckpointCache.Get(prevHash); ok {
			return data.(*bc.Hash), nil
		}

		if prevHeight%state.BlocksOfEpoch == 0 {
			c.prevCheckpointCache.Add(blockHash, &prevHash)
			return &prevHash, nil
		}

		blockHash = &prevHash
	}
}
