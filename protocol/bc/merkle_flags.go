package bc

import (
	"github.com/tendermint/tmlibs/common"
)

// MerkleFlag represent the type of merkle tree node, it's used to generate the structure of merkle tree
// Bitcoin has only two flags, which zero means the hash of assist node. And one means the hash of the related
// transaction node or it's parents, which distinguish them according to the height of the tree. But in the bytom,
// the height of transaction node is not fixed, so we need three flags to distinguish these nodes.
type MerkleFlag uint8

const (
	// FlagAssist represent assist node
	FlagAssist = iota
	// FlagTxParent represent the parent of transaction of node
	FlagTxParent
	// FlagTxLeaf represent transaction of node
	FlagTxLeaf
)

// MerkleFlags contains all flags of one merkle tree
type MerkleFlags struct {
	bitArray *common.BitArray
}

// NewMerkleFlags initialization the bit flags
func NewMerkleFlags(flags []uint8) *MerkleFlags {
	merkleFlags := &MerkleFlags{}
	bitArray := common.NewBitArray(len(flags))
	for i, flag := range flags {
		firstIdx, secondIdx := i*2, i*2+1
		switch flag {
		case FlagAssist:
			bitArray.SetIndex(firstIdx, false)
			bitArray.SetIndex(secondIdx, false)
		case FlagTxParent:
			bitArray.SetIndex(firstIdx, false)
			bitArray.SetIndex(secondIdx, true)
		case FlagTxLeaf:
			bitArray.SetIndex(firstIdx, true)
			bitArray.SetIndex(secondIdx, false)
		}
	}
	merkleFlags.bitArray = bitArray
	return merkleFlags
}

// Bytes return a byte array contains all bits
func (m *MerkleFlags) Bytes() []byte {
	return m.bitArray.Bytes()
}

// GetIndex return the flag at the specify position
func (m *MerkleFlags) GetIndex(index int) MerkleFlag {
	first := m.bitArray.GetIndex(index * 2)
	second := m.bitArray.GetIndex(index*2 + 1)
	if !first && !second {
		return FlagAssist
	} else if !first && second {
		return FlagTxParent
	}
	return FlagTxLeaf
}

// Size return the count of flags
func (m *MerkleFlags) Size() int {
	return m.bitArray.Size() / 2
}
