package protocol

import (
	"math/big"
	"sync"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// BlockNode represents a block within the block chain and is primarily used to
// aid in selecting the best chain to be the main chain.
type BlockNode struct {
	parent  *BlockNode // parent is the parent block for this node.
	hash    bc.Hash    // hash of the block.
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

func NewBlockNode(bh *types.BlockHeader, parent *BlockNode) *BlockNode {
	node := &BlockNode{
		parent:    parent,
		hash:      bh.Hash(),
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
	} else if bh.Height%consensus.SeedPerRetarget == 0 {
		node.seed = &parent.hash
	} else {
		node.seed = parent.seed
	}

	if parent != nil {
		node.workSum = node.workSum.Add(parent.workSum, node.workSum)
	}
	return node
}

func (node *BlockNode) blockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		Version:           node.version,
		Height:            node.height,
		PreviousBlockHash: node.parent.hash,
		Timestamp:         node.timestamp,
		Nonce:             node.nonce,
		Bits:              node.bits,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: node.transactionsMerkleRoot,
			TransactionStatusHash:  node.transactionStatusHash,
		},
	}
}

type BlockIndex struct {
	sync.RWMutex

	index     map[bc.Hash]*BlockNode
	mainChain []*BlockNode
}

func NewBlockIndex() *BlockIndex {
	return &BlockIndex{
		index:     make(map[bc.Hash]*BlockNode),
		mainChain: []*BlockNode{},
	}
}

func (bi *BlockIndex) AddNode(node *BlockNode) {
	bi.Lock()
	bi.index[node.hash] = node
	bi.Unlock()
}

func (bi *BlockIndex) LookupNode(hash *bc.Hash) *BlockNode {
	bi.RLock()
	defer bi.RUnlock()
	return bi.index[*hash]
}

func (bi *BlockIndex) InMainchain(hash bc.Hash) bool {
	bi.RLocker()
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

// NodeByHeight returns the block node at the specified height.  Nil will be
// returned if the height does not exist.
func (bi *BlockIndex) NodeByHeight(height uint64) *BlockNode {
	bi.RLocker()
	defer bi.RUnlock()
	return bi.nodeByHeight(height)
}

func (bi *BlockIndex) SetTip(node *BlockNode) {
	bi.Lock()
	bi.Unlock()

	if uint64(len(bi.mainChain)) == node.height {
		bi.mainChain = append(bi.mainChain, node)
	} else {
		bi.mainChain[node.height] = node
	}
}
