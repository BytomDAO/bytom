package bc

import (
	"io"
	"math"

	"github.com/soniakeys/multiset"

	"github.com/bytom/crypto/sha3pool"
)

var (
	leafPrefix     = []byte{0x00}
	interiorPrefix = []byte{0x01}
)

type MerkleNode interface {
	WriteTo(io.Writer) (int64, error)
	String() string
}

func merkleRoot(nodes []MerkleNode) (root Hash, err error) {
	switch {
	case len(nodes) == 0:
		return EmptyStringHash, nil

	case len(nodes) == 1:
		root = leafMerkleHash(nodes[0])
		return root, nil

	default:
		k := prevPowerOfTwo(len(nodes))
		left, err := merkleRoot(nodes[:k])
		if err != nil {
			return root, err
		}

		right, err := merkleRoot(nodes[k:])
		if err != nil {
			return root, err
		}

		root = interiorMerkleHash(&left, &right)
		return root, nil
	}
}

func interiorMerkleHash(left MerkleNode, right MerkleNode) (hash Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(interiorPrefix)
	left.WriteTo(h)
	right.WriteTo(h)
	hash.ReadFrom(h)
	return hash
}

func leafMerkleHash(node MerkleNode) (hash Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(leafPrefix)
	node.WriteTo(h)
	hash.ReadFrom(h)
	return hash
}

type merkleTreeNode struct {
	MerkleHash *Hash
	rawData    MerkleNode
	left       *merkleTreeNode
	right      *merkleTreeNode
}

// BuildMerkleTree construct a merkle tree based on the provide node data
func BuildMerkleTree(rawDatas []MerkleNode) *merkleTreeNode {
	switch len(rawDatas) {
	case 0:
		return nil
	case 1:
		rawData := rawDatas[0]
		merkleHash := leafMerkleHash(rawData)
		node := newMerkleTreeNode(&merkleHash, rawData, nil, nil)
		return node
	default:
		k := prevPowerOfTwo(len(rawDatas))
		left := BuildMerkleTree(rawDatas[:k])
		right := BuildMerkleTree(rawDatas[k:])
		merkleHash := interiorMerkleHash(left.MerkleHash, right.MerkleHash)
		node := newMerkleTreeNode(&merkleHash, nil, left, right)
		return node
	}
}

func (node *merkleTreeNode) getMerkleTreeProof(rawHashSet multiset.Multiset) (bool, []Hash, []uint8) {
	var hashes []Hash
	var flags []uint8

	if node.left == nil && node.right == nil {
		key := node.rawData.String()
		if rawHashSet.Contains(key, 1) {
			hashes = append(hashes, *node.MerkleHash)
			flags = append(flags, FlagTxLeaf)
			rawHashSet.AssignCount(key, rawHashSet[key]-1)
			return true, hashes, flags
		}
		return false, hashes, flags
	}
	leftFind, leftHashes, leftFlags := node.left.getMerkleTreeProof(rawHashSet)
	rightFind, rightHashes, rightFlags := node.right.getMerkleTreeProof(rawHashSet)

	find := leftFind || rightFind
	if find {
		flags = append(flags, FlagTxParent)
	} else {
		flags = append(flags, FlagAssist)
		hashes = append(hashes, *node.MerkleHash)
		return false, hashes, flags
	}

	if leftFind {
		hashes = append(hashes, leftHashes...)
		flags = append(flags, leftFlags...)
	} else {
		hashes = append(hashes, *node.left.MerkleHash)
		flags = append(flags, FlagAssist)
	}

	if rightFind {
		hashes = append(hashes, rightHashes...)
		flags = append(flags, rightFlags...)
	} else {
		hashes = append(hashes, *node.right.MerkleHash)
		flags = append(flags, FlagAssist)
	}
	return find, hashes, flags
}

func getMerkleTreeProof(rawDatas []MerkleNode, relatedRawDatas []MerkleNode) (bool, []Hash, []uint8) {
	merkleTree := BuildMerkleTree(rawDatas)
	if merkleTree == nil {
		return false, nil, nil
	}
	rawHashSet := multiset.Multiset{}
	for _, data := range relatedRawDatas {
		rawHashSet.AddElements(data.String())
	}
	return merkleTree.getMerkleTreeProof(rawHashSet)
}

// GetTxMerkleTreeProof return a proof of merkle tree, which used to proof the transaction does
// exist in the merkle tree
func GetTxMerkleTreeProof(txIDs []Hash, relatedTxIDs []Hash) (bool, []Hash, []uint8) {
	var rawDatas []MerkleNode
	var relatedRawDatas []MerkleNode
	for _, txID := range txIDs {
		temp := txID
		rawDatas = append(rawDatas, &temp)
	}
	for _, txID := range relatedTxIDs {
		temp := txID
		relatedRawDatas = append(relatedRawDatas, &temp)
	}
	return getMerkleTreeProof(rawDatas, relatedRawDatas)
}

// GetStatusMerkleTreeProof return a proof of merkle tree, which used to proof the status of transaction is valid
func GetStatusMerkleTreeProof(statuses []*TxVerifyResult, relatedStatuses []*TxVerifyResult) (bool, []Hash, []uint8) {
	var rawDatas []MerkleNode
	var relatedRawDatas []MerkleNode
	for _, status := range statuses {
		rawDatas = append(rawDatas, status)
	}
	for _, status := range relatedStatuses {
		relatedRawDatas = append(relatedRawDatas, status)
	}
	return getMerkleTreeProof(rawDatas, relatedRawDatas)
}

// getMerkleRootByProof caculate the merkle root hash according to the proof
func getMerkleRootByProof(hashesPtr *[]Hash, flagsPtr *[]uint8, merkleHashes multiset.Multiset) Hash {
	hashes := *hashesPtr
	flags := *flagsPtr
	if len(flags) == 0 {
		return EmptyStringHash
	}
	flag := flags[0]
	nextFlags := flags[1:]
	*flagsPtr = nextFlags
	if flag == FlagAssist {
		nextHashes := hashes[1:]
		*hashesPtr = nextHashes
		return hashes[0]
	}
	if flag == FlagTxLeaf {
		key := hashes[0].String()
		if len(hashes) != 0 && merkleHashes.Contains(key, 1) {
			nextHashes := hashes[1:]
			*hashesPtr = nextHashes
			merkleHashes.AssignCount(key, merkleHashes[key]-1)
			return hashes[0]
		}
		return EmptyStringHash
	}
	leftHash := getMerkleRootByProof(hashesPtr, flagsPtr, merkleHashes)
	rightHash := getMerkleRootByProof(hashesPtr, flagsPtr, merkleHashes)
	hash := interiorMerkleHash(&leftHash, &rightHash)
	return hash
}

func newMerkleTreeNode(merkleHash *Hash, rawData MerkleNode, left *merkleTreeNode, right *merkleTreeNode) *merkleTreeNode {
	return &merkleTreeNode{
		MerkleHash: merkleHash,
		rawData:    rawData,
		left:       left,
		right:      right,
	}
}

// ValidateMerkleTreeProof caculate the merkle root according to the hash of node and the flags
// only if the merkle root by caculated equals to the specify merkle root, and the merkle tree
// contains all of the related raw datas, the validate result will be true.
func validateMerkleTreeProof(hashes []Hash, flags []uint8, relatedNodes []MerkleNode, merkleRoot Hash) bool {
	merkleHashes := multiset.Multiset{}
	for _, relatedNode := range relatedNodes {
		merkleHash := leafMerkleHash(relatedNode)
		merkleHashes.AddElements(merkleHash.String())
	}
	root := getMerkleRootByProof(&hashes, &flags, merkleHashes)
	return root.String() == merkleRoot.String() && len(merkleHashes) == 0
}

// ValidateTxMerkleTreeProof validate the merkle tree of transactions
func ValidateTxMerkleTreeProof(hashes []Hash, flags []uint8, relatedHashes []Hash, merkleRoot Hash) bool {
	var relatedNodes []MerkleNode
	for _, hash := range relatedHashes {
		temp := hash
		relatedNodes = append(relatedNodes, &temp)
	}
	return validateMerkleTreeProof(hashes, flags, relatedNodes, merkleRoot)
}

// ValidateStatusMerkleTreeProof validate the merkle tree of transaction status
func ValidateStatusMerkleTreeProof(hashes []Hash, flags []uint8, relatedStatus []*TxVerifyResult, merkleRoot Hash) bool {
	var relatedNodes []MerkleNode
	for _, result := range relatedStatus {
		relatedNodes = append(relatedNodes, result)
	}
	return validateMerkleTreeProof(hashes, flags, relatedNodes, merkleRoot)
}

// TxStatusMerkleRoot creates a merkle tree from a slice of TxVerifyResult
func TxStatusMerkleRoot(tvrs []*TxVerifyResult) (root Hash, err error) {
	nodes := []MerkleNode{}
	for _, tvr := range tvrs {
		nodes = append(nodes, tvr)
	}
	return merkleRoot(nodes)
}

// TxMerkleRoot creates a merkle tree from a slice of transactions
// and returns the root hash of the tree.
func TxMerkleRoot(transactions []*Tx) (root Hash, err error) {
	nodes := []MerkleNode{}
	for _, tx := range transactions {
		nodes = append(nodes, &tx.ID)
	}
	return merkleRoot(nodes)
}

// prevPowerOfTwo returns the largest power of two that is smaller than a given number.
// In other words, for some input n, the prevPowerOfTwo k is a power of two such that
// k < n <= 2k. This is a helper function used during the calculation of a merkle tree.
func prevPowerOfTwo(n int) int {
	// If the number is a power of two, divide it by 2 and return.
	if n&(n-1) == 0 {
		return n / 2
	}

	// Otherwise, find the previous PoT.
	exponent := uint(math.Log2(float64(n)))
	return 1 << exponent // 2^exponent
}
