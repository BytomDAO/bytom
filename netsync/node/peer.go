package node

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

	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
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

func (p *peer) getStatus() (height uint64, hash *bc.Hash) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.height, p.hash
}

func (p *peer) setStatus(height uint64, hash *bc.Hash) {
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

func (p *peer) sendTransactions(txs []*types.Tx) error {
	for _, tx := range txs {
		msg, err := NewTransactionNotifyMessage(tx)
		if err != nil {
			return errors.New("Failed construction tx msg")
		}
		hash := &tx.ID
		p.knownTxs.Add(hash.String())
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
func (p *peer) markTransaction(hash *bc.Hash) {
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
func (p *peer) markBlock(hash *bc.Hash) {
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

type PeerSet struct {
	peers  map[string]*peer
	sw     *p2p.Switch
	lock   sync.RWMutex
	closed bool
}

// NewPeerSet creates a new peer set to track the active participants.
func NewPeerSet(sw *p2p.Switch) *PeerSet {
	return &PeerSet{
		sw:    sw,
		peers: make(map[string]*peer),
	}
}

// Len returns if the current number of peers in the set.
func (ps *PeerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (ps *PeerSet) MarkTransaction(peerID string, hash *bc.Hash) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.markTransaction(hash)
	}
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (ps *PeerSet) MarkBlock(peerID string, hash *bc.Hash) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.markBlock(hash)
	}
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *PeerSet) PeersWithoutBlock(hash *bc.Hash) []*peer {
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
func (ps *PeerSet) PeersWithoutTx(hash *bc.Hash) []*peer {
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

func (ps *PeerSet) AddBanScore(peerID string, persistent, transient uint64, reason string) bool {
	ps.lock.Lock()

	peer, ok := ps.peers[peerID]
	if !ok {
		ps.lock.Unlock()
		return true
	}

	ban := peer.addBanScore(persistent, transient, reason)
	if ban {
		ps.sw.AddBannedPeer(peer.getPeer())
		ps.initiativeRemovePeer(peer)
	} else {
		ps.lock.Unlock()
	}
	return ban
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *PeerSet) BestPeer() (*p2p.Peer, uint64) {
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
func (ps *PeerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.swPeer.CloseConn()
	}
	ps.closed = true
}

func (ps *PeerSet) AddPeer(peer *p2p.Peer) {
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

func (ps *PeerSet) RemovePeer(peerID string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.peers, peerID)
	log.WithField("ID", peerID).Info("Delete peer from peerset")
}

func (ps *PeerSet) InitiativeRemovePeer(peerID string) {
	ps.lock.Lock()

	peer, ok := ps.peers[peerID]
	if !ok {
		ps.lock.Unlock()
		return
	}

	ps.initiativeRemovePeer(peer)
}

func (ps *PeerSet) initiativeRemovePeer(peer *peer) {
	delete(ps.peers, peer.id)
	log.WithField("ID", peer.id).Info("Delete peer from peerset")
	ps.lock.Unlock()
	ps.sw.StopPeerGracefully(peer.getPeer())
}

func (ps *PeerSet) SetPeerStatus(peerID string, height uint64, hash *bc.Hash) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.setStatus(height, hash)
	}
}

func (ps *PeerSet) requestBlockByHash(peerID string, hash *bc.Hash) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("Can't find peer. ")
	}
	return peer.requestBlockByHash(hash)
}

func (ps *PeerSet) RequestBlockByHeight(peerID string, height uint64) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("Can't find peer. ")
	}
	return peer.requestBlockByHeight(height)
}

func (ps *PeerSet) SendTransactions(peerID string, txs []*types.Tx) error {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("Can't find peer. ")
	}

	return peer.sendTransactions(txs)
}

func (ps *PeerSet) BroadcastMinedBlock(block *types.Block) error {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return errors.New("Failed construction block msg")
	}
	hash := block.Hash()
	peers := ps.PeersWithoutBlock(&hash)
	for _, peer := range peers {
		if ok := peer.swPeer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			peer.addBanScore(0, 50, "Broadcast block error")
			continue
		}
		if p, ok := ps.peers[peer.id]; ok {
			p.markBlock(&hash)
		}
	}
	return nil
}

func (ps *PeerSet) BroadcastNewStatus(block *types.Block) error {
	return ps.BroadcastMinedBlock(block)
}

func (ps *PeerSet) BroadcastTx(tx *types.Tx) error {
	msg, err := NewTransactionNotifyMessage(tx)
	if err != nil {
		return errors.New("Failed construction tx msg")
	}
	peers := ps.PeersWithoutTx(&tx.ID)
	for _, peer := range peers {
		if ok := peer.swPeer.Send(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
			peer.addBanScore(0, 50, "Broadcast new tx error")
			continue
		}
		if p, ok := ps.peers[peer.id]; ok {
			p.markTransaction(&tx.ID)
		}
	}
	return nil
}
