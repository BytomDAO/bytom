package consensus

import (
	"sync"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

var (
	errVerifySignature          = errors.New("signature of verification message is invalid")
	errPubKeyIsNotValidator     = errors.New("pub key is not in validators of target checkpoint")
	errVoteToGrowingCheckpoint  = errors.New("validator publish vote to growing checkpoint")
	errSameHeightInVerification = errors.New("validator publish two distinct votes for the same target height")
	errSpanHeightInVerification = errors.New("validator publish vote within the span of its other votes")
)

// Casper is BFT based proof of stack consensus algorithm, it provides safety and liveness in theory,
// it's design mainly refers to https://github.com/ethereum/research/blob/master/papers/casper-basics/casper_basics.pdf
type Casper struct {
	mu    sync.RWMutex
	tree  *treeNode
	store protocol.Store
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

	c.mu.RLock()
	defer c.mu.RUnlock()

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
	if err := v.VerifySignature(); err != nil {
		return err
	}

	source, err := c.store.GetCheckpoint(&v.SourceHash)
	if err != nil {
		// the synchronization block is later than the arrival of the verification message
		return err
	}

	target, err := c.store.GetCheckpoint(&v.TargetHash)
	if err != nil {
		return err
	}

	if source.Status == state.Growing || target.Status == state.Growing {
		return errVoteToGrowingCheckpoint
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !target.ContainsValidator(v.PubKey) {
		return errPubKeyIsNotValidator
	}

	if err := c.verifySameHeight(v); err != nil {
		return err
	}

	if err := c.verifySpanHeight(v); err != nil {
		return err
	}

	supLink := target.AddSupLink(v.SourceHeight, v.SourceHash, v.PubKey)
	if source.Status == state.Justified && supLink.Confirmed() {
		target.Status = state.Justified
		source.Status = state.Finalized
		// must notify chain when rollback
		if err := c.pruneTree(source.Hash); err != nil {
			return err
		}
	}
	return nil
}

func (c *Casper) pruneTree(rootHash bc.Hash) error {
	newRoot, err := c.tree.nodeByHash(rootHash)
	if err != nil {
		return err
	}

	c.tree = newRoot
	return nil
}

// ProcessBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
func (c *Casper) ProcessBlock(block *types.Block) (*state.Checkpoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prevHash := block.PreviousBlockHash
	node, err := c.tree.nodeByHash(prevHash)
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
			Mortgages:      make(map[string]uint64),
		}
		node.children = append(node.children, &treeNode{checkpoint: checkpoint})
	} else if mod == 0 {
		checkpoint.Status = state.Unverified
	}

	checkpoint.Height = block.Height
	checkpoint.Hash = block.Hash()

	for range block.Transactions {
		// process the votes and mortgages
	}
	return checkpoint, nil
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *Verification) error {
	checkpoints, err := c.store.GetCheckpointsByHeight(v.TargetHeight)
	if err != nil {
		return err
	}

	for _, checkpoint := range checkpoints {
		for _, supLink := range checkpoint.SupLinks {
			if supLink.PubKeys[v.PubKey] {
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *Verification) error {
	// It is best to scan all checkpoints from leveldb
	if c.tree.findOnlyOne(func(c *state.Checkpoint) bool {
		if c.Height <= v.TargetHeight {
			return false
		}

		for _, supLink := range c.SupLinks {
			if supLink.PubKeys[v.PubKey] && supLink.SourceHeight < v.SourceHeight {
				return true
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
	if checkpoint.Status == state.Justified || checkpoint.Status == state.Finalized {
		justifiedHeight = checkpoint.Height
	}

	bestHeight, bestHash, maxJustifiedHeight := checkpoint.Height, checkpoint.Hash, justifiedHeight
	for _, child := range node.children {
		if height, hash, justified := chainOfMaxJustifiedHeight(child, justifiedHeight); justified > maxJustifiedHeight {
			bestHeight, bestHash, maxJustifiedHeight = height, hash, justified
		}
	}
	return bestHeight, bestHash, maxJustifiedHeight
}

func (c *Casper) prevCheckpointHash(blockHash *bc.Hash) (*bc.Hash, error) {
	for {
		block, err := c.store.GetBlockHeader(blockHash)
		if err != nil {
			return nil, err
		}

		height := block.Height - 1
		blockHash = &block.PreviousBlockHash
		if height%state.BlocksOfEpoch == 0 {
			return blockHash, nil
		}
	}
}
