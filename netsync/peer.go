package netsync

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/p2p/trust"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	defaultVersion      = 1
	defaultBanThreshold = uint64(100)
)

type peer struct {
	mtx      sync.RWMutex
	version  int // Protocol version negotiated
	id       string
	height   uint64
	hash     *bc.Hash
	banScore trust.DynamicBanScore

	swPeer *p2p.Peer

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(height uint64, hash *bc.Hash, Peer *p2p.Peer) *peer {
	return &peer{
		version:     defaultVersion,
		id:          Peer.Key,
		height:      height,
		hash:        hash,
		swPeer:      Peer,
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
	p.swPeer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	return nil
}

func (p *peer) requestBlockByHeight(height uint64) error {
	msg := &BlockRequestMessage{Height: height}
	p.swPeer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
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
		if p.swPeer == nil {
			return errPeerDropped
		}
		p.swPeer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg})
	}
	return nil
}

func (p *peer) getPeer() *p2p.Peer {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.swPeer
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

// addBanScore increases the persistent and decaying ban score fields by the
// values passed as parameters. If the resulting score exceeds half of the ban
// threshold, a warning is logged including the reason provided. Further, if
// the score is above the ban threshold, the peer will be banned and
// disconnected.
func (p *peer) addBanScore(persistent, transient uint64, reason string) bool {
	warnThreshold := defaultBanThreshold >> 1
	if transient == 0 && persistent == 0 {
		// The score is not being increased, but a warning message is still
		// logged if the score is above the warn threshold.
		score := p.banScore.Int()
		if score > warnThreshold {
			log.Infof("Misbehaving peer %s: %s -- ban score is %d, "+"it was not increased this time", p.id, reason, score)
		}
		return false
	}
	score := p.banScore.Increase(persistent, transient)
	if score > warnThreshold {
		log.Infof("Misbehaving peer %s: %s -- ban score increased to %d", p.id, reason, score)
		if score > defaultBanThreshold {
			log.Errorf("Misbehaving peer %s -- banning and disconnecting", p.id)
			return true
		}
	}
	return false
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
func (ps *peerSet) Peer(id string) (*peer, bool) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	p, ok := ps.peers[id]
	return p, ok
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
			bestPeer, bestHeight = p.swPeer, p.height
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
		p.swPeer.CloseConn()
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
	peer, ok := ps.Peer(peerID)
	if !ok {
		return errors.New("Can't find peer. ")
	}
	return peer.requestBlockByHash(hash)
}

func (ps *peerSet) requestBlockByHeight(peerID string, height uint64) error {
	peer, ok := ps.Peer(peerID)
	if !ok {
		return errors.New("Can't find peer. ")
	}
	return peer.requestBlockByHeight(height)
}

func (ps *peerSet) BroadcastMinedBlock(block *types.Block) ([]*peer, error) {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return nil, errors.New("Failed construction block msg")
	}
	hash := block.Hash()
	peers := ps.PeersWithoutBlock(&hash)
	abnormalPeers := make([]*peer, 0)
	for _, peer := range peers {
		if ok := peer.swPeer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			abnormalPeers = append(abnormalPeers, peer)
			continue
		}
		if p, ok := ps.Peer(peer.id); ok {
			p.MarkBlock(&hash)
		}
	}
	return abnormalPeers, nil
}

func (ps *peerSet) BroadcastNewStatus(block *types.Block) ([]*peer, error) {
	return ps.BroadcastMinedBlock(block)
}

func (ps *peerSet) BroadcastTx(tx *types.Tx) ([]*peer, error) {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return nil, errors.New("Failed construction tx msg")
	}
	peers := ps.PeersWithoutTx(&tx.ID)
	abnormalPeers := make([]*peer, 0)
	for _, peer := range peers {
		if ok := peer.swPeer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			abnormalPeers = append(abnormalPeers, peer)
			continue
		}
		if p, ok := ps.Peer(peer.id); ok {
			p.MarkTransaction(&tx.ID)
		}
	}
	return abnormalPeers, nil
}
