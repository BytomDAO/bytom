package iavl

import (
	"bytes"
	"io"

	"golang.org/x/crypto/ripemd160"

	. "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/go-wire"
)

// Node

type IAVLNode struct {
	key       []byte
	value     []byte
	height    int8
	size      int
	hash      []byte
	leftHash  []byte
	leftNode  *IAVLNode
	rightHash []byte
	rightNode *IAVLNode
	persisted bool
}

func NewIAVLNode(key []byte, value []byte) *IAVLNode {
	return &IAVLNode{
		key:    key,
		value:  value,
		height: 0,
		size:   1,
	}
}

// NOTE: The hash is not saved or set.  The caller should set the hash afterwards.
// (Presumably the caller already has the hash)
func MakeIAVLNode(buf []byte, t *IAVLTree) (node *IAVLNode, err error) {
	node = &IAVLNode{}

	// node header
	node.height = int8(buf[0])
	buf = buf[1:]
	var n int
	node.size, n, err = wire.GetVarint(buf)
	if err != nil {
		return nil, err
	}
	buf = buf[n:]
	node.key, n, err = wire.GetByteSlice(buf)
	if err != nil {
		return nil, err
	}
	buf = buf[n:]
	if node.height == 0 {
		// value
		node.value, n, err = wire.GetByteSlice(buf)
		if err != nil {
			return nil, err
		}
		buf = buf[n:]
	} else {
		// children
		leftHash, n, err := wire.GetByteSlice(buf)
		if err != nil {
			return nil, err
		}
		buf = buf[n:]
		rightHash, n, err := wire.GetByteSlice(buf)
		if err != nil {
			return nil, err
		}
		buf = buf[n:]
		node.leftHash = leftHash
		node.rightHash = rightHash
	}
	return node, nil
}

func (node *IAVLNode) _copy() *IAVLNode {
	if node.height == 0 {
		PanicSanity("Why are you copying a value node?")
	}
	return &IAVLNode{
		key:       node.key,
		height:    node.height,
		size:      node.size,
		hash:      nil, // Going to be mutated anyways.
		leftHash:  node.leftHash,
		leftNode:  node.leftNode,
		rightHash: node.rightHash,
		rightNode: node.rightNode,
		persisted: false, // Going to be mutated, so it can't already be persisted.
	}
}

func (node *IAVLNode) has(t *IAVLTree, key []byte) (has bool) {
	if bytes.Compare(node.key, key) == 0 {
		return true
	}
	if node.height == 0 {
		return false
	} else {
		if bytes.Compare(key, node.key) < 0 {
			return node.getLeftNode(t).has(t, key)
		} else {
			return node.getRightNode(t).has(t, key)
		}
	}
}

func (node *IAVLNode) get(t *IAVLTree, key []byte) (index int, value []byte, exists bool) {
	if node.height == 0 {
		cmp := bytes.Compare(node.key, key)
		if cmp == 0 {
			return 0, node.value, true
		} else if cmp == -1 {
			return 1, nil, false
		} else {
			return 0, nil, false
		}
	} else {
		if bytes.Compare(key, node.key) < 0 {
			return node.getLeftNode(t).get(t, key)
		} else {
			rightNode := node.getRightNode(t)
			index, value, exists = rightNode.get(t, key)
			index += node.size - rightNode.size
			return index, value, exists
		}
	}
}

func (node *IAVLNode) getByIndex(t *IAVLTree, index int) (key []byte, value []byte) {
	if node.height == 0 {
		if index == 0 {
			return node.key, node.value
		} else {
			PanicSanity("getByIndex asked for invalid index")
			return nil, nil
		}
	} else {
		// TODO: could improve this by storing the
		// sizes as well as left/right hash.
		leftNode := node.getLeftNode(t)
		if index < leftNode.size {
			return leftNode.getByIndex(t, index)
		} else {
			return node.getRightNode(t).getByIndex(t, index-leftNode.size)
		}
	}
}

// NOTE: sets hashes recursively
func (node *IAVLNode) hashWithCount(t *IAVLTree) ([]byte, int) {
	if node.hash != nil {
		return node.hash, 0
	}

	hasher := ripemd160.New()
	buf := new(bytes.Buffer)
	_, hashCount, err := node.writeHashBytes(t, buf)
	if err != nil {
		PanicCrisis(err)
	}
	hasher.Write(buf.Bytes())
	node.hash = hasher.Sum(nil)

	return node.hash, hashCount + 1
}

// NOTE: sets hashes recursively
func (node *IAVLNode) writeHashBytes(t *IAVLTree, w io.Writer) (n int, hashCount int, err error) {
	// height & size
	wire.WriteInt8(node.height, w, &n, &err)
	wire.WriteVarint(node.size, w, &n, &err)
	// key is not written for inner nodes, unlike writePersistBytes

	if node.height == 0 {
		// key & value
		wire.WriteByteSlice(node.key, w, &n, &err)
		wire.WriteByteSlice(node.value, w, &n, &err)
	} else {
		// left
		if node.leftNode != nil {
			leftHash, leftCount := node.leftNode.hashWithCount(t)
			node.leftHash = leftHash
			hashCount += leftCount
		}
		if node.leftHash == nil {
			PanicSanity("node.leftHash was nil in writeHashBytes")
		}
		wire.WriteByteSlice(node.leftHash, w, &n, &err)
		// right
		if node.rightNode != nil {
			rightHash, rightCount := node.rightNode.hashWithCount(t)
			node.rightHash = rightHash
			hashCount += rightCount
		}
		if node.rightHash == nil {
			PanicSanity("node.rightHash was nil in writeHashBytes")
		}
		wire.WriteByteSlice(node.rightHash, w, &n, &err)
	}
	return
}

// NOTE: clears leftNode/rigthNode recursively
// NOTE: sets hashes recursively
func (node *IAVLNode) save(t *IAVLTree) {
	if node.hash == nil {
		node.hash, _ = node.hashWithCount(t)
	}
	if node.persisted {
		return
	}

	// save children
	if node.leftNode != nil {
		node.leftNode.save(t)
		node.leftNode = nil
	}
	if node.rightNode != nil {
		node.rightNode.save(t)
		node.rightNode = nil
	}

	// save node
	t.ndb.SaveNode(t, node)
	return
}

// NOTE: sets hashes recursively
func (node *IAVLNode) writePersistBytes(t *IAVLTree, w io.Writer) (n int, err error) {
	// node header
	wire.WriteInt8(node.height, w, &n, &err)
	wire.WriteVarint(node.size, w, &n, &err)
	// key (unlike writeHashBytes, key is written for inner nodes)
	wire.WriteByteSlice(node.key, w, &n, &err)

	if node.height == 0 {
		// value
		wire.WriteByteSlice(node.value, w, &n, &err)
	} else {
		// left
		if node.leftHash == nil {
			PanicSanity("node.leftHash was nil in writePersistBytes")
		}
		wire.WriteByteSlice(node.leftHash, w, &n, &err)
		// right
		if node.rightHash == nil {
			PanicSanity("node.rightHash was nil in writePersistBytes")
		}
		wire.WriteByteSlice(node.rightHash, w, &n, &err)
	}
	return
}

func (node *IAVLNode) set(t *IAVLTree, key []byte, value []byte) (newSelf *IAVLNode, updated bool) {
	if node.height == 0 {
		cmp := bytes.Compare(key, node.key)
		if cmp < 0 {
			return &IAVLNode{
				key:       node.key,
				height:    1,
				size:      2,
				leftNode:  NewIAVLNode(key, value),
				rightNode: node,
			}, false
		} else if cmp == 0 {
			removeOrphan(t, node)
			return NewIAVLNode(key, value), true
		} else {
			return &IAVLNode{
				key:       key,
				height:    1,
				size:      2,
				leftNode:  node,
				rightNode: NewIAVLNode(key, value),
			}, false
		}
	} else {
		removeOrphan(t, node)
		node = node._copy()
		if bytes.Compare(key, node.key) < 0 {
			node.leftNode, updated = node.getLeftNode(t).set(t, key, value)
			node.leftHash = nil // leftHash is yet unknown
		} else {
			node.rightNode, updated = node.getRightNode(t).set(t, key, value)
			node.rightHash = nil // rightHash is yet unknown
		}
		if updated {
			return node, updated
		} else {
			node.calcHeightAndSize(t)
			return node.balance(t), updated
		}
	}
}

// newHash/newNode: The new hash or node to replace node after remove.
// newKey: new leftmost leaf key for tree after successfully removing 'key' if changed.
// value: removed value.
func (node *IAVLNode) remove(t *IAVLTree, key []byte) (
	newHash []byte, newNode *IAVLNode, newKey []byte, value []byte, removed bool) {
	if node.height == 0 {
		if bytes.Compare(key, node.key) == 0 {
			removeOrphan(t, node)
			return nil, nil, nil, node.value, true
		} else {
			return node.hash, node, nil, nil, false
		}
	} else {
		if bytes.Compare(key, node.key) < 0 {
			var newLeftHash []byte
			var newLeftNode *IAVLNode
			newLeftHash, newLeftNode, newKey, value, removed = node.getLeftNode(t).remove(t, key)
			if !removed {
				return node.hash, node, nil, value, false
			} else if newLeftHash == nil && newLeftNode == nil { // left node held value, was removed
				return node.rightHash, node.rightNode, node.key, value, true
			}
			removeOrphan(t, node)
			node = node._copy()
			node.leftHash, node.leftNode = newLeftHash, newLeftNode
			node.calcHeightAndSize(t)
			node = node.balance(t)
			return node.hash, node, newKey, value, true
		} else {
			var newRightHash []byte
			var newRightNode *IAVLNode
			newRightHash, newRightNode, newKey, value, removed = node.getRightNode(t).remove(t, key)
			if !removed {
				return node.hash, node, nil, value, false
			} else if newRightHash == nil && newRightNode == nil { // right node held value, was removed
				return node.leftHash, node.leftNode, nil, value, true
			}
			removeOrphan(t, node)
			node = node._copy()
			node.rightHash, node.rightNode = newRightHash, newRightNode
			if newKey != nil {
				node.key = newKey
			}
			node.calcHeightAndSize(t)
			node = node.balance(t)
			return node.hash, node, nil, value, true
		}
	}
}

func (node *IAVLNode) getLeftNode(t *IAVLTree) *IAVLNode {
	if node.leftNode != nil {
		return node.leftNode
	} else {
		return t.ndb.GetNode(t, node.leftHash)
	}
}

func (node *IAVLNode) getRightNode(t *IAVLTree) *IAVLNode {
	if node.rightNode != nil {
		return node.rightNode
	} else {
		return t.ndb.GetNode(t, node.rightHash)
	}
}

// NOTE: overwrites node
// TODO: optimize balance & rotate
func (node *IAVLNode) rotateRight(t *IAVLTree) *IAVLNode {
	node = node._copy()
	l := node.getLeftNode(t)
	removeOrphan(t, l)
	_l := l._copy()

	_lrHash, _lrCached := _l.rightHash, _l.rightNode
	_l.rightHash, _l.rightNode = node.hash, node
	node.leftHash, node.leftNode = _lrHash, _lrCached

	node.calcHeightAndSize(t)
	_l.calcHeightAndSize(t)

	return _l
}

// NOTE: overwrites node
// TODO: optimize balance & rotate
func (node *IAVLNode) rotateLeft(t *IAVLTree) *IAVLNode {
	node = node._copy()
	r := node.getRightNode(t)
	removeOrphan(t, r)
	_r := r._copy()

	_rlHash, _rlCached := _r.leftHash, _r.leftNode
	_r.leftHash, _r.leftNode = node.hash, node
	node.rightHash, node.rightNode = _rlHash, _rlCached

	node.calcHeightAndSize(t)
	_r.calcHeightAndSize(t)

	return _r
}

// NOTE: mutates height and size
func (node *IAVLNode) calcHeightAndSize(t *IAVLTree) {
	node.height = maxInt8(node.getLeftNode(t).height, node.getRightNode(t).height) + 1
	node.size = node.getLeftNode(t).size + node.getRightNode(t).size
}

func (node *IAVLNode) calcBalance(t *IAVLTree) int {
	return int(node.getLeftNode(t).height) - int(node.getRightNode(t).height)
}

// NOTE: assumes that node can be modified
// TODO: optimize balance & rotate
func (node *IAVLNode) balance(t *IAVLTree) (newSelf *IAVLNode) {
	if node.persisted {
		panic("Unexpected balance() call on persisted node")
	}
	balance := node.calcBalance(t)
	if balance > 1 {
		if node.getLeftNode(t).calcBalance(t) >= 0 {
			// Left Left Case
			return node.rotateRight(t)
		} else {
			// Left Right Case
			// node = node._copy()
			left := node.getLeftNode(t)
			removeOrphan(t, left)
			node.leftHash, node.leftNode = nil, left.rotateLeft(t)
			//node.calcHeightAndSize()
			return node.rotateRight(t)
		}
	}
	if balance < -1 {
		if node.getRightNode(t).calcBalance(t) <= 0 {
			// Right Right Case
			return node.rotateLeft(t)
		} else {
			// Right Left Case
			// node = node._copy()
			right := node.getRightNode(t)
			removeOrphan(t, right)
			node.rightHash, node.rightNode = nil, right.rotateRight(t)
			//node.calcHeightAndSize()
			return node.rotateLeft(t)
		}
	}
	// Nothing changed
	return node
}

// traverse is a wrapper over traverseInRange when we want the whole tree
func (node *IAVLNode) traverse(t *IAVLTree, ascending bool, cb func(*IAVLNode) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, cb)
}

func (node *IAVLNode) traverseInRange(t *IAVLTree, start, end []byte, ascending bool, cb func(*IAVLNode) bool) bool {
	afterStart := (start == nil || bytes.Compare(start, node.key) <= 0)
	beforeEnd := (end == nil || bytes.Compare(node.key, end) <= 0)

	stop := false
	if afterStart && beforeEnd {
		// IterateRange ignores this if not leaf
		stop = cb(node)
	}
	if stop {
		return stop
	}

	if node.height > 0 {
		if ascending {
			// check lower nodes, then higher
			if afterStart {
				stop = node.getLeftNode(t).traverseInRange(t, start, end, ascending, cb)
			}
			if stop {
				return stop
			}
			if beforeEnd {
				stop = node.getRightNode(t).traverseInRange(t, start, end, ascending, cb)
			}
		} else {
			// check the higher nodes first
			if beforeEnd {
				stop = node.getRightNode(t).traverseInRange(t, start, end, ascending, cb)
			}
			if stop {
				return stop
			}
			if afterStart {
				stop = node.getLeftNode(t).traverseInRange(t, start, end, ascending, cb)
			}
		}
	}

	return stop
}

// Only used in testing...
func (node *IAVLNode) lmd(t *IAVLTree) *IAVLNode {
	if node.height == 0 {
		return node
	}
	return node.getLeftNode(t).lmd(t)
}

// Only used in testing...
func (node *IAVLNode) rmd(t *IAVLTree) *IAVLNode {
	if node.height == 0 {
		return node
	}
	return node.getRightNode(t).rmd(t)
}

//----------------------------------------

func removeOrphan(t *IAVLTree, node *IAVLNode) {
	if !node.persisted {
		return
	}
	if t.ndb == nil {
		return
	}
	t.ndb.RemoveNode(t, node)
}
