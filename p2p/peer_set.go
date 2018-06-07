package p2p

import (
	"sync"
)

// IPeerSet has a (immutable) subset of the methods of PeerSet.
type IPeerSet interface {
	Has(key string) bool
	Get(key string) *Peer
	List() []*Peer
	Size() int
}

//-----------------------------------------------------------------------------

// PeerSet is a special structure for keeping a table of peers.
// Iteration over the peers is super fast and thread-safe.
type PeerSet struct {
	mtx    sync.Mutex
	lookup map[string]*peerSetItem
	list   []*Peer
}

type peerSetItem struct {
	peer  *Peer
	index int
}

// NewPeerSet creates a new peerSet with a list of initial capacity of 256 items.
func NewPeerSet() *PeerSet {
	return &PeerSet{
		lookup: make(map[string]*peerSetItem),
		list:   make([]*Peer, 0, 256),
	}
}

// Add adds the peer to the PeerSet.
// Returns false if peer with key (PubKeyEd25519) is already set
func (ps *PeerSet) Add(peer *Peer) error {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.lookup[peer.Key] != nil {
		return ErrDuplicatePeer
	}

	ps.lookup[peer.Key] = &peerSetItem{peer, len(ps.list)}
	ps.list = append(ps.list, peer)
	return nil
}

// Get looks up a peer by the provided peerKey.
func (ps *PeerSet) Get(peerKey string) *Peer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	item, ok := ps.lookup[peerKey]
	if ok {
		return item.peer
	}
	return nil
}

// Has returns true if the PeerSet contains
// the peer referred to by this peerKey.
func (ps *PeerSet) Has(peerKey string) bool {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	_, ok := ps.lookup[peerKey]
	return ok
}

// List threadsafe list of peers.
func (ps *PeerSet) List() []*Peer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.list
}

// Remove discards peer if the peer was previously memoized.
func (ps *PeerSet) Remove(peer *Peer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	item := ps.lookup[peer.Key]
	if item == nil {
		return
	}

	index := item.index
	// Copy the list but without the last element.
	// (we must copy because we're mutating the list)
	newList := make([]*Peer, len(ps.list)-1)
	copy(newList, ps.list)
	// If it's the last peer, that's an easy special case.
	if index == len(ps.list)-1 {
		ps.list = newList
		delete(ps.lookup, peer.Key)
		return
	}

	// Move the last item from ps.list to "index" in list.
	lastPeer := ps.list[len(ps.list)-1]
	lastPeerKey := lastPeer.Key
	lastPeerItem := ps.lookup[lastPeerKey]
	newList[index] = lastPeer
	lastPeerItem.index = index
	ps.list = newList
	delete(ps.lookup, peer.Key)
}

// Size returns the number of unique items in the peerSet.
func (ps *PeerSet) Size() int {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return len(ps.list)
}
