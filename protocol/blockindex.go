package protocol

import (
	"errors"
	"math/big"
	"sort"
	"sync"

	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// approxNodesPerDay is an approximation of the number of new blocks there are
// in a week on average.
const approxNodesPerDay = 24 * 24

// BlockNode represents a block within the block chain and is primarily used to
// aid in selecting the best chain to be the main chain.
type BlockNode struct {
	parent  *BlockNode // parent is the parent block for this node.
	Hash    bc.Hash    // hash of the block.
	seed    *bc.Hash   // seed hash of the block
	workSum *big.Int   // total amount of work in the chain up to

	version                uint64
	height                 uint64
	timestamp              uint64
	nonce                  uint64
	bits                   uint64
	transactionsMerkleRoot bc.Hash
	transactionStatusHash  bc.Hash
}

func NewBlockNode(bh *types.BlockHeader, parent *BlockNode) (*BlockNode, error) {
	if bh.Height != 0 && parent == nil {
		return nil, errors.New("parent node can not be nil")
	}

	node := &BlockNode{
		parent:    parent,
		Hash:      bh.Hash(),
		workSum:   difficulty.CalcWork(bh.Bits),
		version:   bh.Version,
		height:    bh.Height,
		timestamp: bh.Timestamp,
		nonce:     bh.Nonce,
		bits:      bh.Bits,
		transactionsMerkleRoot: bh.TransactionsMerkleRoot,
		transactionStatusHash:  bh.TransactionStatusHash,
	}

	if bh.Height == 0 {
		node.seed = consensus.InitialSeed
	} else {
		node.seed = parent.CalcNextSeed()
		node.workSum = node.workSum.Add(parent.workSum, node.workSum)
	}
	return node, nil
}

// blockHeader convert a node to the header struct
func (node *BlockNode) blockHeader() *types.BlockHeader {
	previousBlockHash := bc.Hash{}
	if node.parent != nil {
		previousBlockHash = node.parent.Hash
	}
	return &types.BlockHeader{
		Version:           node.version,
		Height:            node.height,
		PreviousBlockHash: previousBlockHash,
		Timestamp:         node.timestamp,
		Nonce:             node.nonce,
		Bits:              node.bits,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: node.transactionsMerkleRoot,
			TransactionStatusHash:  node.transactionStatusHash,
		},
	}
}

func (node *BlockNode) CalcPastMedianTime() uint64 {
	timestamps := []uint64{}
	iterNode := node
	for i := 0; i < consensus.MedianTimeBlocks && iterNode != nil; i++ {
		timestamps = append(timestamps, iterNode.timestamp)
		iterNode = iterNode.parent
	}

	sort.Sort(common.TimeSorter(timestamps))
	return timestamps[len(timestamps)/2]
}

// CalcNextBits calculate the seed for next block
func (node *BlockNode) CalcNextBits() uint64 {
	if node.height%consensus.BlocksPerRetarget != 0 || node.height == 0 {
		return node.bits
	}

	compareNode := node.parent
	for compareNode.height%consensus.BlocksPerRetarget != 0 {
		compareNode = compareNode.parent
	}
	return difficulty.CalcNextRequiredDifficulty(node.blockHeader(), compareNode.blockHeader())
}

// CalcNextSeed calculate the seed for next block
func (node *BlockNode) CalcNextSeed() *bc.Hash {
	if node.height%consensus.SeedPerRetarget == 0 {
		return &node.Hash
	}
	return node.seed
}

// BlockIndex is the struct for help chain trace block chain as tree
type BlockIndex struct {
	sync.RWMutex

	index     map[bc.Hash]*BlockNode
	mainChain []*BlockNode
}

// NewBlockIndex will create a empty BlockIndex
func NewBlockIndex() *BlockIndex {
	return &BlockIndex{
		index:     make(map[bc.Hash]*BlockNode),
		mainChain: make([]*BlockNode, 0, approxNodesPerDay),
	}
}

// AddNode will add node to the index map
func (bi *BlockIndex) AddNode(node *BlockNode) {
	bi.Lock()
	bi.index[node.Hash] = node
	bi.Unlock()
}

// GetNode will search node from the index map
func (bi *BlockIndex) GetNode(hash *bc.Hash) *BlockNode {
	bi.RLock()
	defer bi.RUnlock()
	return bi.index[*hash]
}

func (bi *BlockIndex) BestNode() *BlockNode {
	bi.RLock()
	defer bi.RUnlock()
	return bi.mainChain[len(bi.mainChain)-1]
}

// BlockExist check does the block existed in blockIndex
func (bi *BlockIndex) BlockExist(hash *bc.Hash) bool {
	bi.RLock()
	_, ok := bi.index[*hash]
	bi.RUnlock()
	return ok
}

// TODO: THIS FUNCTION MIGHT BE DELETED
func (bi *BlockIndex) InMainchain(hash bc.Hash) bool {
	bi.RLock()
	defer bi.RUnlock()

	node, ok := bi.index[hash]
	if !ok {
		return false
	}
	return bi.nodeByHeight(node.height) == node
}

func (bi *BlockIndex) nodeByHeight(height uint64) *BlockNode {
	if height >= uint64(len(bi.mainChain)) {
		return nil
	}
	return bi.mainChain[height]
}

// NodeByHeight returns the block node at the specified height.
func (bi *BlockIndex) NodeByHeight(height uint64) *BlockNode {
	bi.RLock()
	defer bi.RUnlock()
	return bi.nodeByHeight(height)
}

// SetMainChain will set the the mainChain array
func (bi *BlockIndex) SetMainChain(node *BlockNode) {
	bi.Lock()
	defer bi.Unlock()

	needed := node.height + 1
	if uint64(cap(bi.mainChain)) < needed {
		nodes := make([]*BlockNode, needed, needed+approxNodesPerDay)
		copy(nodes, bi.mainChain)
		bi.mainChain = nodes
	} else {
		i := uint64(len(bi.mainChain))
		bi.mainChain = bi.mainChain[0:needed]
		for ; i < needed; i++ {
			bi.mainChain[i] = nil
		}
	}

	for node != nil && bi.mainChain[node.height] != node {
		bi.mainChain[node.height] = node
		node = node.parent
	}
}
