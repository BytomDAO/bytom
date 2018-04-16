package netsync

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const defaultVersion = 1

type peer struct {
	mtx     sync.RWMutex
	version int // Protocol version negotiated
	id      string
	height  uint64
	hash    *bc.Hash
	*p2p.Peer

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(height uint64, hash *bc.Hash, Peer *p2p.Peer) *peer {
	return &peer{
		version:     defaultVersion,
		height:      height,
		hash:        hash,
		Peer:        Peer,
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

func (p *peer) GetStatus() (height uint64, hash *bc.Hash) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.height, p.hash
}

func (p *peer) SetStatus(height uint64, hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.height = height
	p.hash = hash
}

func (p *peer) requestBlockByHash(hash *bc.Hash) error {
	msg := &BlockRequestMessage{RawHash: hash.Byte32()}
	p.Peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (p *peer) requestBlockByHeight(height uint64) error {
	msg := &BlockRequestMessage{Height: height}
	p.Peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (p *peer) SendTransactions(txs []*types.Tx) error {
	for _, tx := range txs {
		msg, err := NewTransactionNotifyMessage(tx)
		if err != nil {
			return errors.New("Failed construction tx msg")
		}
		hash := &tx.ID
		p.knownTxs.Add(hash.String())
		p.Peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	}
	return nil
}

func (p *peer) getPeer() *p2p.Peer {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.Peer
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash.String())
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) MarkBlock(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash.String())
}

type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (ps *peerSet) MarkTransaction(peerID string, hash *bc.Hash) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.MarkTransaction(hash)
	}
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (ps *peerSet) MarkBlock(peerID string, hash *bc.Hash) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.MarkBlock(hash)
	}
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutBlock(hash *bc.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownBlocks.Has(hash.String()) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash *bc.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownTxs.Has(hash.String()) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() (*p2p.Peer, uint64) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var bestPeer *p2p.Peer
	var bestHeight uint64

	for _, p := range ps.peers {
		if bestPeer == nil || p.height > bestHeight {
			bestPeer, bestHeight = p.Peer, p.height
		}
	}

	return bestPeer, bestHeight
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.CloseConn()
	}
	ps.closed = true
}

func (ps *peerSet) AddPeer(peer *p2p.Peer) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[peer.Key]; !ok {
		keeperPeer := newPeer(0, nil, peer)
		ps.peers[peer.Key] = keeperPeer
		log.WithFields(log.Fields{"ID": peer.Key}).Info("Add new peer to blockKeeper")
		return
	}
	log.WithField("ID", peer.Key).Warning("Add existing peer to blockKeeper")
}

func (ps *peerSet) RemovePeer(peerID string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.peers, peerID)
	log.WithField("ID", peerID).Info("Delete peer from peerset")
}

func (ps *peerSet) SetPeerStatus(peerID string, height uint64, hash *bc.Hash) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.SetStatus(height, hash)
	}
}

func (ps *peerSet) requestBlockByHash(peerID string, hash *bc.Hash) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("Can't find peer. ")
	}
	return peer.requestBlockByHash(hash)
}

func (ps *peerSet) requestBlockByHeight(peerID string, height uint64) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("Can't find peer. ")
	}
	return peer.requestBlockByHeight(height)
}

func (ps *peerSet) BroadcastMinedBlock(block *types.Block) error {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return errors.New("Failed construction block msg")
	}
	hash := block.Hash()
	peers := ps.PeersWithoutBlock(&hash)
	for _, peer := range peers {
		ps.MarkBlock(peer.Key, &hash)
		peer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg})
	}
	return nil
}

func (ps *peerSet) BroadcastNewStatus(block *types.Block) error {
	return ps.BroadcastMinedBlock(block)
}

func (ps *peerSet) BroadcastTx(tx *types.Tx) error {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return errors.New("Failed construction tx msg")
	}
	peers := ps.PeersWithoutTx(&tx.ID)
	for _, peer := range peers {
		ps.peers[peer.Key].MarkTransaction(&tx.ID)
		peer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg})
	}
	return nil
}
