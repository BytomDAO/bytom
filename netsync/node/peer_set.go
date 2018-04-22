package node

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

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

func (ps *PeerSet) RequestBlockByHash(peerID string, hash *bc.Hash) error {
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
		return nil
	}

	for _, tx := range txs {
		peer.sendTransaction(tx)
	}
	return nil
}

func (ps *PeerSet) SendBlock(peerID string, block *types.Block) bool {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return true
	}

	return peer.sendBlock(block)
}

func (ps *PeerSet) BroadcastMinedBlock(block *types.Block) error {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, peer := range ps.peers {
		if ok := peer.sendBlock(block); !ok {
			peer.addBanScore(0, 50, "Broadcast block error")
			continue
		}
	}
	return nil
}

func (ps *PeerSet) BroadcastTx(tx *types.Tx) error {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, peer := range ps.peers {
		if ok := peer.sendTransaction(tx); !ok {
			peer.addBanScore(0, 50, "Broadcast new tx error")
			continue
		}
	}
	return nil
}
