package netsync

import (
	"errors"
	"sync"

	"gopkg.in/fatih/set.v0"

	"github.com/bytom/p2p"
	"github.com/bytom/protocol/bc"
)

type blockKeeperPeer struct {
	mtx    sync.RWMutex
	height uint64
	hash   *bc.Hash
	peer   *p2p.Peer

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newBlockKeeperPeer(height uint64, hash *bc.Hash) *blockKeeperPeer {
	return &blockKeeperPeer{
		height:      height,
		hash:        hash,
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

func (p *blockKeeperPeer) GetStatus() (height uint64, hash *bc.Hash) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.height, p.hash
}

func (p *blockKeeperPeer) SetStatus(height uint64, hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.height = height
	p.hash = hash
}

func (bkp *blockKeeperPeer) requestBlockByHash(hash *bc.Hash) error {
	peer := bkp.peer
	if peer == nil {
		return errors.New("Can't find peer.")
	}
	msg := &BlockRequestMessage{RawHash: hash.Byte32()}
	peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (bkp *blockKeeperPeer) requestBlockByHeight(height uint64) error {
	peer := bkp.peer
	if peer == nil {
		return errors.New("Can't find peer.")
	}
	msg := &BlockRequestMessage{Height: height}
	peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (bkp *blockKeeperPeer) MarkTransaction(hash [32]byte) {
	bkp.mtx.Lock()
	defer bkp.mtx.Unlock()

	// If we reached the memory allowance, drop a previously known transaction hash
	for bkp.knownTxs.Size() >= maxKnownTxs {
		bkp.knownTxs.Pop()
	}
	bkp.knownTxs.Add(hash)
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (bkp *blockKeeperPeer) MarkBlock(hash [32]byte) {
	bkp.mtx.Lock()
	defer bkp.mtx.Unlock()

	// If we reached the memory allowance, drop a previously known block hash
	for bkp.knownBlocks.Size() >= maxKnownBlocks {
		bkp.knownBlocks.Pop()
	}
	bkp.knownBlocks.Add(hash)
}
