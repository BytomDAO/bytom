package types

import (
	"container/list"
	"io"
	"math"

	"gopkg.in/fatih/set.v0"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
)

// merkleFlag represent the type of merkle tree node, it's used to generate the structure of merkle tree
// Bitcoin has only two flags, which zero means the hash of assist node. And one means the hash of the related
// transaction node or it's parents, which distinguish them according to the height of the tree. But in the bytom,
// the height of transaction node is not fixed, so we need three flags to distinguish these nodes.
const (
	// FlagAssist represent assist node
	FlagAssist = iota
	// FlagTxParent represent the parent of transaction of node
	FlagTxParent
	// FlagTxLeaf represent transaction of node
	FlagTxLeaf
)

var (
	leafPrefix     = []byte{0x00}
	interiorPrefix = []byte{0x01}
)

type merkleNode interface {
	WriteTo(io.Writer) (int64, error)
}

func merkleRoot(nodes []merkleNode) (root bc.Hash, err error) {
	switch {
	case len(nodes) == 0:
		return bc.EmptyStringHash, nil

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

func interiorMerkleHash(left merkleNode, right merkleNode) (hash bc.Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(interiorPrefix)
	left.WriteTo(h)
	right.WriteTo(h)
	hash.ReadFrom(h)
	return hash
}

func leafMerkleHash(node merkleNode) (hash bc.Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(leafPrefix)
	node.WriteTo(h)
	hash.ReadFrom(h)
	return hash
}

type merkleTreeNode struct {
	hash  bc.Hash
	left  *merkleTreeNode
	right *merkleTreeNode
}

// buildMerkleTree construct a merkle tree based on the provide node data
func buildMerkleTree(rawDatas []merkleNode) *merkleTreeNode {
	switch len(rawDatas) {
	case 0:
		return nil
	case 1:
		rawData := rawDatas[0]
		merkleHash := leafMerkleHash(rawData)
		node := newMerkleTreeNode(merkleHash, nil, nil)
		return node
	default:
		k := prevPowerOfTwo(len(rawDatas))
		left := buildMerkleTree(rawDatas[:k])
		right := buildMerkleTree(rawDatas[k:])
		merkleHash := interiorMerkleHash(&left.hash, &right.hash)
		node := newMerkleTreeNode(merkleHash, left, right)
		return node
	}
}

func (node *merkleTreeNode) getMerkleTreeProof(merkleHashSet *set.Set) ([]*bc.Hash, []uint8) {
	var hashes []*bc.Hash
	var flags []uint8

	if node.left == nil && node.right == nil {
		if key := node.hash.String(); merkleHashSet.Has(key) {
			hashes = append(hashes, &node.hash)
			flags = append(flags, FlagTxLeaf)
			return hashes, flags
		}
		return hashes, flags
	}
	var leftHashes, rightHashes []*bc.Hash
	var leftFlags, rightFlags []uint8
	if node.left != nil {
		leftHashes, leftFlags = node.left.getMerkleTreeProof(merkleHashSet)
	}
	if node.right != nil {
		rightHashes, rightFlags = node.right.getMerkleTreeProof(merkleHashSet)
	}
	leftFind, rightFind := len(leftHashes) > 0, len(rightHashes) > 0

	if leftFind || rightFind {
		flags = append(flags, FlagTxParent)
	} else {
		return hashes, flags
	}

	if leftFind {
		hashes = append(hashes, leftHashes...)
		flags = append(flags, leftFlags...)
	} else {
		hashes = append(hashes, &node.left.hash)
		flags = append(flags, FlagAssist)
	}

	if rightFind {
		hashes = append(hashes, rightHashes...)
		flags = append(flags, rightFlags...)
	} else {
		hashes = append(hashes, &node.right.hash)
		flags = append(flags, FlagAssist)
	}
	return hashes, flags
}

func getMerkleTreeProof(rawDatas []merkleNode, relatedRawDatas []merkleNode) ([]*bc.Hash, []uint8) {
	merkleTree := buildMerkleTree(rawDatas)
	if merkleTree == nil {
		return []*bc.Hash{}, []uint8{}
	}
	merkleHashSet := set.New()
	for _, data := range relatedRawDatas {
		merkleHash := leafMerkleHash(data)
		merkleHashSet.Add(merkleHash.String())
	}
	return merkleTree.getMerkleTreeProof(merkleHashSet)
}

func (node *merkleTreeNode) getMerkleTreeProofByFlags(flagList *list.List) []*bc.Hash {
	var hashes []*bc.Hash

	if flagList.Len() == 0 {
		return hashes
	}
	flagEle := flagList.Front()
	flag := flagEle.Value.(uint8)
	flagList.Remove(flagEle)

	if flag == FlagTxLeaf || flag == FlagAssist {
		hashes = append(hashes, &node.hash)
		return hashes
	}
	if node.left != nil {
		leftHashes := node.left.getMerkleTreeProofByFlags(flagList)
		hashes = append(hashes, leftHashes...)
	}
	if node.right != nil {
		rightHashes := node.right.getMerkleTreeProofByFlags(flagList)
		hashes = append(hashes, rightHashes...)
	}
	return hashes
}

func getMerkleTreeProofByFlags(rawDatas []merkleNode, flagList *list.List) []*bc.Hash {
	tree := buildMerkleTree(rawDatas)
	return tree.getMerkleTreeProofByFlags(flagList)
}

// GetTxMerkleTreeProof return a proof of merkle tree, which used to proof the transaction does
// exist in the merkle tree
func GetTxMerkleTreeProof(txs []*Tx, relatedTxs []*Tx) ([]*bc.Hash, []uint8) {
	var rawDatas []merkleNode
	var relatedRawDatas []merkleNode
	for _, tx := range txs {
		rawDatas = append(rawDatas, &tx.ID)
	}
	for _, relatedTx := range relatedTxs {
		relatedRawDatas = append(relatedRawDatas, &relatedTx.ID)
	}
	return getMerkleTreeProof(rawDatas, relatedRawDatas)
}

// GetStatusMerkleTreeProof return a proof of merkle tree, which used to proof the status of transaction is valid
func GetStatusMerkleTreeProof(statuses []*bc.TxVerifyResult, flags []uint8) []*bc.Hash {
	var rawDatas []merkleNode
	for _, status := range statuses {
		rawDatas = append(rawDatas, status)
	}
	flagList := list.New()
	for _, flag := range flags {
		flagList.PushBack(flag)
	}
	return getMerkleTreeProofByFlags(rawDatas, flagList)
}

// getMerkleRootByProof caculate the merkle root hash according to the proof
func getMerkleRootByProof(hashList *list.List, flagList *list.List, merkleHashes *list.List) bc.Hash {
	if flagList.Len() == 0 {
		return bc.EmptyStringHash
	}
	flagEle := flagList.Front()
	flag := flagEle.Value.(uint8)
	flagList.Remove(flagEle)
	if flag == FlagAssist {
		hash := hashList.Front()
		hashList.Remove(hash)
		return hash.Value.(bc.Hash)
	}
	if flag == FlagTxLeaf {
		if hashList.Len() == 0 || merkleHashes.Len() == 0 {
			return bc.EmptyStringHash
		}
		hashEle := hashList.Front()
		hash := hashEle.Value.(bc.Hash)
		relatedHashEle := merkleHashes.Front()
		relatedHash := relatedHashEle.Value.(bc.Hash)
		if hash == relatedHash {
			hashList.Remove(hashEle)
			merkleHashes.Remove(relatedHashEle)
			return hash
		}
		return bc.EmptyStringHash
	}
	leftHash := getMerkleRootByProof(hashList, flagList, merkleHashes)
	rightHash := getMerkleRootByProof(hashList, flagList, merkleHashes)
	hash := interiorMerkleHash(&leftHash, &rightHash)
	return hash
}

func newMerkleTreeNode(merkleHash bc.Hash, left *merkleTreeNode, right *merkleTreeNode) *merkleTreeNode {
	return &merkleTreeNode{
		hash:  merkleHash,
		left:  left,
		right: right,
	}
}

// ValidateMerkleTreeProof caculate the merkle root according to the hash of node and the flags
// only if the merkle root by caculated equals to the specify merkle root, and the merkle tree
// contains all of the related raw datas, the validate result will be true.
func validateMerkleTreeProof(hashes []*bc.Hash, flags []uint8, relatedNodes []merkleNode, merkleRoot bc.Hash) bool {
	merkleHashes := list.New()
	for _, relatedNode := range relatedNodes {
		merkleHashes.PushBack(leafMerkleHash(relatedNode))
	}
	hashList := list.New()
	for _, hash := range hashes {
		hashList.PushBack(*hash)
	}
	flagList := list.New()
	for _, flag := range flags {
		flagList.PushBack(flag)
	}
	root := getMerkleRootByProof(hashList, flagList, merkleHashes)
	return root == merkleRoot && merkleHashes.Len() == 0
}

// ValidateTxMerkleTreeProof validate the merkle tree of transactions
func ValidateTxMerkleTreeProof(hashes []*bc.Hash, flags []uint8, relatedHashes []*bc.Hash, merkleRoot bc.Hash) bool {
	var relatedNodes []merkleNode
	for _, hash := range relatedHashes {
		relatedNodes = append(relatedNodes, hash)
	}
	return validateMerkleTreeProof(hashes, flags, relatedNodes, merkleRoot)
}

// ValidateStatusMerkleTreeProof validate the merkle tree of transaction status
func ValidateStatusMerkleTreeProof(hashes []*bc.Hash, flags []uint8, relatedStatus []*bc.TxVerifyResult, merkleRoot bc.Hash) bool {
	var relatedNodes []merkleNode
	for _, result := range relatedStatus {
		relatedNodes = append(relatedNodes, result)
	}
	return validateMerkleTreeProof(hashes, flags, relatedNodes, merkleRoot)
}

// TxStatusMerkleRoot creates a merkle tree from a slice of bc.TxVerifyResult
func TxStatusMerkleRoot(tvrs []*bc.TxVerifyResult) (root bc.Hash, err error) {
	nodes := []merkleNode{}
	for _, tvr := range tvrs {
		nodes = append(nodes, tvr)
	}
	return merkleRoot(nodes)
}

// TxMerkleRoot creates a merkle tree from a slice of transactions
// and returns the root hash of the tree.
func TxMerkleRoot(transactions []*bc.Tx) (root bc.Hash, err error) {
	nodes := []merkleNode{}
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
