package consensus

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"sync"

	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

var (
	errVerifySignature          = errors.New("signature of verification message is invalid")
	errPubKeyIsNotValidator     = errors.New("pub key is not in validators of target checkpoint")
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
func (c *Casper) Validators(blockHash *bc.Hash) ([]*Validator, error) {
	hash, err := c.prevCheckpointHash(blockHash)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	checkpoint, err := c.tree.checkpointByHash(hash.String())
	if err != nil {
		return nil, err
	}

	return checkpoint.validators(), nil
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

// EncodeMessage encode the verification for the validators to sign or verify
func (v *Verification) EncodeMessage() ([]byte, error) {
	buff := new(bytes.Buffer)
	if _, err := buff.WriteString(v.SourceHash); err != nil {
		return nil, err
	}

	if _, err := buff.WriteString(v.TargetHash); err != nil {
		return nil, err
	}

	uint64Buff := make([]byte, 8)

	binary.LittleEndian.PutUint64(uint64Buff, v.SourceHeight)
	if _, err := buff.Write(uint64Buff); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint64(uint64Buff, v.TargetHeight)
	if _, err := buff.Write(uint64Buff); err != nil {
		return nil, err
	}

	return sha3Hash(buff.Bytes())
}

// VerifySignature verify the signature of encode message of verification
func (v *Verification) VerifySignature() error {
	pubKey, err := hex.DecodeString(v.PubKey)
	if err != nil {
		return err
	}

	signature, err := hex.DecodeString(v.Signature)
	if err != nil {
		return err
	}

	message, err := v.EncodeMessage()
	if err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, message, signature) {
		return errVerifySignature
	}

	return nil
}

// AuthVerification verify whether the Verification is legal.
// the status of source checkpoint must justified, and an individual validator ν must not publish two distinct Verification
// ⟨ν,s1,t1,h(s1),h(t1)⟩ and ⟨ν,s2,t2,h(s2),h(t2)⟩, such that either:
// h(t1) = h(t2) OR h(s1) < h(s2) < h(t2) < h(t1)
func (c *Casper) AuthVerification(v *Verification) error {
	if err := v.VerifySignature(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	source, err := c.tree.checkpointByHash(v.SourceHash)
	if err != nil {
		// the following two cases are not handled
		// case1: the synchronization block is later than the arrival of the verification message
		// case2: the tree node was was pruned
		return err
	}

	target, err := c.tree.checkpointByHash(v.TargetHash)
	if err != nil {
		return err
	}

	if target.isValidator(v.PubKey) {
		return errPubKeyIsNotValidator
	}

	if err := c.verifySameHeight(v); err != nil {
		return err
	}

	if err := c.verifySpanHeight(v); err != nil {
		return err
	}

	supLink := target.addSupLink(v.SourceHeight, v.SourceHash, v.PubKey)
	if source.status == justified && supLink.confirmed() {
		target.status = justified
		source.status = finalized
		// must notify chain when rollback
		// pruning the tree
	}
	return nil
}

// ProcessBlock used to receive a new block from upper layer, it provides idempotence
// and parse the vote and mortgage from the transactions, then save to the checkpoint
// the tree of checkpoint will grow with the arrival of new blocks
func (c *Casper) ProcessBlock(block *types.Block) error {
	return nil
}

// a validator must not publish two distinct votes for the same target height
func (c *Casper) verifySameHeight(v *Verification) error {
	nodes := c.tree.checkpointsOfHeight(v.TargetHeight)
	for _, node := range nodes {
		for _, supLink := range node.supLinks {
			if supLink.pubKeys[v.PubKey] {
				return errSameHeightInVerification
			}
		}
	}
	return nil
}

// a validator must not vote within the span of its other votes.
func (c *Casper) verifySpanHeight(v *Verification) error {
	if c.tree.findOnlyOne(func(c *checkpoint) bool {
		if c.height <= v.TargetHeight {
			return false
		}

		for _, supLink := range c.supLinks {
			if supLink.pubKeys[v.PubKey] && supLink.sourceHeight < v.SourceHeight {
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

func (c *Casper) prevCheckpointHash(blockHash *bc.Hash) (*bc.Hash, error) {
	for {
		block, err := c.store.GetBlockHeader(blockHash)
		if err != nil {
			return nil, err
		}

		height := block.Height - 1
		blockHash = &block.PreviousBlockHash
		if height%blocksOfEpoch == 0 {
			return blockHash, nil
		}
	}
}

func sha3Hash(message []byte) ([]byte, error) {
	sha3 := sha3pool.Get256()
	defer sha3pool.Put256(sha3)

	if _, err := sha3.Write(message); err != nil {
		return nil, err
	}

	hash := &bc.Hash{}
	if _, err := hash.ReadFrom(sha3); err != nil {
		return nil, err
	}

	return hash.Bytes(), nil
}
