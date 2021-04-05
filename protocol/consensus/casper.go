package consensus

import (
	"sync"

	"github.com/bytom/bytom/protocol/bc/types"
)

type treeNode struct {
	checkpoint *checkpoint
	children   []*treeNode
}

// Casper is BFT based proof of stack consensus algorithm, it provides safety and liveness in theory,
// it's design mainly refers to https://github.com/ethereum/research/blob/master/papers/casper-basics/casper_basics.pdf
type Casper struct {
	mu   sync.RWMutex
	tree *treeNode
}

// Best chain return the chain containing the justified checkpoint of the largest height
func (c *Casper) BestChain() (uint64, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// root is init justified
	root := c.tree.checkpoint
	bestHeight, bestHash, _ := chainOfMaxJustifiedHeight(c.tree, root.height)
	return bestHeight, bestHash
}

// Validators return the validators by specified block hash
// e.g. if the block num of epoch is 100, and the block height corresponding to the block hash is 130, then will return the voting results of height in 0~100
func (c *Casper) Validators(blockHash string) []*Validator {
	return nil
}

// Verification represent a verification message for the block
// source hash and target hash point to the checkpoint, and the source checkpoint is the target checkpoint's parent(not be directly)
// the vector <sourceHash, targetHash, sourceHeight, targetHeight, pubKey> as the message of signature
type Verification struct {
	SourceHash   string
	TargetHash   string
	SourceHeight uint64
	TargetHeight uint64
	Signature    string
	PubKey       string
}

// AuthVerification verify whether the Verification is legal.
// the status of source checkpoint must justified, and an individual validator ν must not publish two distinct Verification
// ⟨ν,s1,t1,h(s1),h(t1)⟩ and ⟨ν,s2,t2,h(s2),h(t2)⟩, such that either:
// h(t1) = h(t2) OR h(s1) < h(s2) < h(t2) < h(t1)
func (c *Casper) AuthVerification(v *Verification) error {
	return nil
}

// ProcessBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
func (c *Casper) ProcessBlock(block *types.Block) error {
	return nil
}

// justifiedHeight is the max justified height of checkpoint from node to root
func chainOfMaxJustifiedHeight(node *treeNode, justifiedHeight uint64) (uint64, string, uint64) {
	checkpoint := node.checkpoint
	if checkpoint.status == justified || checkpoint.status == finalized {
		justifiedHeight = checkpoint.height
	}

	bestHeight, bestHash, maxJustifiedHeight := checkpoint.height, checkpoint.hash, justifiedHeight
	for _, child := range node.children {
		if height, hash, justified := chainOfMaxJustifiedHeight(child, justifiedHeight); justified > maxJustifiedHeight {
			bestHeight, bestHash, maxJustifiedHeight = height, hash, justified
		}
	}
	return bestHeight, bestHash, maxJustifiedHeight
}
