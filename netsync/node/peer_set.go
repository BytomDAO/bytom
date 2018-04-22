package node

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// PeerSet contains all the connected peer info
type PeerSet struct {
	peers map[string]*peer
	sw    *p2p.Switch
	lock  sync.RWMutex
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

// AddBanScore will add score to the peer & dc the peer if require
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

func (ps *PeerSet) initiativeRemovePeer(peer *peer) {
	delete(ps.peers, peer.id)
	log.WithField("ID", peer.id).Info("initiative remove peer from peer set")
	ps.lock.Unlock()
	ps.sw.StopPeerGracefully(peer.getPeer())
}

// InitiativeRemovePeer disconnect a peer
func (ps *PeerSet) InitiativeRemovePeer(peerID string) {
	ps.lock.Lock()

	peer, ok := ps.peers[peerID]
	if !ok {
		ps.lock.Unlock()
		return
	}

	ps.initiativeRemovePeer(peer)
}

// SetPeerStatus will set peer height and hash
func (ps *PeerSet) SetPeerStatus(peerID string, height uint64, hash *bc.Hash) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	if peer, ok := ps.peers[peerID]; ok {
		peer.setStatus(height, hash)
	}
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *PeerSet) BestPeer() (string, uint64) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var best *peer
	for _, p := range ps.peers {
		if p.height > best.height {
			best = p
		}
	}

	return best.id, best.height
}

// AddPeer add a new peer to thec peer set
func (ps *PeerSet) AddPeer(peer *p2p.Peer, height uint64, hash *bc.Hash) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[peer.Key]; !ok {
		log.WithField("ID", peer.Key).Warning("add existing peer to peer set")
		return
	}

	ps.peers[peer.Key] = newPeer(height, hash, peer)
	log.WithFields(log.Fields{"ID": peer.Key}).Info("add new peer to peer set")
}

// RemovePeer handle the dc action from switch level
func (ps *PeerSet) RemovePeer(peerID string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	delete(ps.peers, peerID)
	log.WithField("ID", peerID).Info("Delete peer from peer set")
}

// ReqBlockByHeight req the block from selected peer
func (ps *PeerSet) ReqBlockByHeight(peerID string, height uint64) error {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("can't find the peer in the peer set")
	}

	if ok = peer.reqBlockByHeight(height); !ok {
		return errors.New("fail to sent req block by height")
	}
	return nil
}

// SendTransactions ...
func (ps *PeerSet) SendTransactions(peerID string, txs []*types.Tx) error {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("can't find the peer in the peer set")
	}

	for _, tx := range txs {
		if ok = peer.sendTransaction(tx); !ok {
			return errors.New("fail to sent req tx")
		}
	}
	return nil
}

// SendBlock ...
func (ps *PeerSet) SendBlock(peerID string, block *types.Block) error {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	peer, ok := ps.peers[peerID]
	if !ok {
		return errors.New("can't find the peer in the peer set")
	}

	if ok = peer.sendBlock(block); !ok {
		return errors.New("fail to sent req")
	}
	return nil
}

// BroadcastMinedBlock ...
func (ps *PeerSet) BroadcastMinedBlock(block *types.Block) {
	ps.lock.RLock()

	banPeers := []string{}
	for _, peer := range ps.peers {
		if ok := peer.sendBlock(block); ok {
			continue
		}
		if ban := peer.addBanScore(0, 50, "Broadcast block error"); ban {
			banPeers = append(banPeers, peer.id)
		}
	}
	ps.lock.RUnlock()

	for _, banPeer := range banPeers {
		ps.InitiativeRemovePeer(banPeer)
	}
	return
}

// BroadcastTx ...
func (ps *PeerSet) BroadcastTx(tx *types.Tx) {
	ps.lock.RLock()

	banPeers := []string{}
	for _, peer := range ps.peers {
		if ok := peer.sendTransaction(tx); ok {
			continue
		}
		if ban := peer.addBanScore(0, 50, "Broadcast new tx error"); ban {
			banPeers = append(banPeers, peer.id)
		}
	}
	ps.lock.RUnlock()

	for _, banPeer := range banPeers {
		ps.InitiativeRemovePeer(banPeer)
	}
	return
}
