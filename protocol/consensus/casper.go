package consensus

import (
	"encoding/hex"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
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
	prvKey           chainkd.XPrv
	// pubKey -> conflicting verifications
	evilValidators map[string][]*Verification
	// block hash -> previous checkpoint hash
	prevCheckpointCache *common.Cache
	// block hash + pubKey -> verification
	verificationCache *common.Cache
}

// NewCasper create a new instance of Casper
// argument checkpoints load the checkpoints from leveldb
// the first element of checkpoints must genesis checkpoint or the last finalized checkpoint in order to reduce memory space
// the others must successors of first one
func NewCasper(store protocol.Store, prvKey chainkd.XPrv, checkpoints []*state.Checkpoint) *Casper {
	if checkpoints[0].Height != 0 && checkpoints[0].Status != state.Finalized {
		log.Panic("first element of checkpoints must genesis or in finalized status")
	}

	casper := &Casper{
		tree:                makeTree(checkpoints[0], checkpoints[1:]),
		rollbackNotifyCh:    make(chan bc.Hash),
		newEpochCh:          make(chan bc.Hash),
		store:               store,
		prvKey:              prvKey,
		evilValidators:      make(map[string][]*Verification),
		prevCheckpointCache: common.NewCache(1024),
		verificationCache:   common.NewCache(1024),
	}
	go casper.authVerificationLoop()
	return casper
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
	hash, err := c.prevCheckpointHash(blockHash)
	if err != nil {
		return nil, err
	}

	checkpoint, err := c.store.GetCheckpoint(hash)
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
	if source.Status == state.Justified && target.Status != state.Justified && supLink.IsMajority() {
		c.setJustified(target)
		// must direct child
		if target.Parent.Hash == source.Hash {
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

	if block.Height % state.BlocksOfEpoch == 0 {
		c.newEpochCh <- block.Hash()
	}

	return c.myVerification(target, validators)
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

func (c *Casper) myVerification(target *state.Checkpoint, validators []*state.Validator) (*Verification, error) {
	pubKey := c.prvKey.XPub().String()
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

		if err := v.Sign(c.prvKey); err != nil {
			return nil, err
		}

		if err := c.verifyVerification(v, false); err != nil {
			return nil, nil
		}

		return v, c.addVerificationToCheckpoint(target, v)
	}
	return nil, nil
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

func isValidator(pubKey string, validators []*state.Validator) bool {
	for _, v := range validators {
		if v.PubKey == pubKey {
			return true
		}
	}
	return false
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
			c.prevCheckpointCache.Add(blockHash, data)
			return data.(*bc.Hash), nil
		}

		if prevHeight%state.BlocksOfEpoch == 0 {
			c.prevCheckpointCache.Add(blockHash, &prevHash)
			return &prevHash, nil
		}

		blockHash = &prevHash
	}
}
