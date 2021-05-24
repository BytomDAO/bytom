package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

var (
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
	rollbackNotifyCh chan interface{}
	newEpochCh       chan bc.Hash
	store            Store
	// pubKey -> conflicting verifications
	evilValidators map[string][]*Verification
	// block hash -> previous checkpoint hash
	prevCheckpointCache *common.Cache
	// block hash + pubKey -> verification
	verificationCache *common.Cache
	// put the checkpoints which exist a majority supLink but the source checkpoint is not justified
	justifyingCheckpoints map[bc.Hash][]*state.Checkpoint
}

// NewCasper create a new instance of Casper
// argument checkpoints load the checkpoints from leveldb
// the first element of checkpoints must genesis checkpoint or the last finalized checkpoint in order to reduce memory space
// the others must be successors of first one
func NewCasper(store Store, checkpoints []*state.Checkpoint, rollbackNotifyCh chan interface{}) *Casper {
	if checkpoints[0].Height != 0 && checkpoints[0].Status != state.Finalized {
		log.Panic("first element of checkpoints must genesis or in finalized status")
	}

	casper := &Casper{
		tree:                  makeTree(checkpoints[0], checkpoints[1:]),
		rollbackNotifyCh:      rollbackNotifyCh,
		newEpochCh:            make(chan bc.Hash),
		store:                 store,
		evilValidators:        make(map[string][]*Verification),
		prevCheckpointCache:   common.NewCache(1024),
		verificationCache:     common.NewCache(1024),
		justifyingCheckpoints: make(map[bc.Hash][]*state.Checkpoint),
	}
	go casper.authVerificationLoop()
	return casper
}

// Best chain return the chain containing the justified checkpoint of the largest height
func (c *Casper) BestChain() (uint64, bc.Hash) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.bestChain()
}

func (c *Casper) bestChain() (uint64, bc.Hash) {
	// root is init justified
	root := c.tree.checkpoint
	bestHeight, bestHash, _ := chainOfMaxJustifiedHeight(c.tree, root.Height)
	return bestHeight, bestHash
}

// LastFinalized return the block height and block hash which is finalized ast last
func (c *Casper) LastFinalized() (uint64, bc.Hash) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	root := c.tree.checkpoint
	return root.Height, root.Hash
}

// Validators return the validators by specified block hash
// e.g. if the block num of epoch is 100, and the block height corresponding to the block hash is 130, then will return the voting results of height in 0~100
func (c *Casper) Validators(blockHash *bc.Hash) (map[string]*state.Validator, error) {
	checkpoint, err := c.prevCheckpoint(blockHash)
	if err != nil {
		return nil, err
	}

	return checkpoint.Validators(), nil
}

func (c *Casper) prevCheckpoint(blockHash *bc.Hash) (*state.Checkpoint, error) {
	hash, err := c.prevCheckpointHash(blockHash)
	if err != nil {
		return nil, err
	}

	return c.store.GetCheckpoint(hash)
}

func (c *Casper) prevCheckpointByPrevHash(prevBlockHash *bc.Hash) (*state.Checkpoint, error) {
	hash, err := c.prevCheckpointHashByPrevHash(prevBlockHash)
	if err != nil {
		return nil, err
	}

	return c.store.GetCheckpoint(hash)
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

func (c *Casper) prevCheckpointHash(blockHash *bc.Hash) (*bc.Hash, error) {
	if data, ok := c.prevCheckpointCache.Get(*blockHash); ok {
		return data.(*bc.Hash), nil
	}

	block, err := c.store.GetBlockHeader(blockHash)
	if err != nil {
		return nil, err
	}

	result, err := c.prevCheckpointHashByPrevHash(&block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	c.prevCheckpointCache.Add(*blockHash, result)
	return result, nil
}

func (c *Casper) prevCheckpointHashByPrevHash(prevBlockHash *bc.Hash) (*bc.Hash, error) {
	prevHash := prevBlockHash
	for {
		if data, ok := c.prevCheckpointCache.Get(*prevHash); ok {
			c.prevCheckpointCache.Add(*prevBlockHash, data)
			return data.(*bc.Hash), nil
		}

		prevBlock, err := c.store.GetBlockHeader(prevHash)
		if err != nil {
			return nil, err
		}

		if prevBlock.Height%state.BlocksOfEpoch == 0 {
			c.prevCheckpointCache.Add(*prevBlockHash, prevHash)
			return prevHash, nil
		}

		prevHash = &prevBlock.PreviousBlockHash
	}
}