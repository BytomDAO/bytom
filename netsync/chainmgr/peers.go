package chainmgr

import (
	"errors"
	"sync"
)

var errNoValidFastSyncPeer = errors.New("no valid fast sync peer")

type fastSyncPeers struct {
	peers map[string]bool
	mtx   sync.RWMutex
}

func newFastSyncPeers() *fastSyncPeers {
	return &fastSyncPeers{
		peers: make(map[string]bool),
	}
}

func (fs *fastSyncPeers) add(peerID string) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	if _, ok := fs.peers[peerID]; ok {
		return
	}

	fs.peers[peerID] = false
}

func (fs *fastSyncPeers) delete(peerID string) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	delete(fs.peers, peerID)
}

func (fs *fastSyncPeers) selectIdlePeers() []string {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	peers := make([]string, 0)
	for peerID, isBusy := range fs.peers {
		if isBusy {
			continue
		}

		fs.peers[peerID] = true
		peers = append(peers, peerID)
	}

	return peers
}

func (fs *fastSyncPeers) selectIdlePeer() (string, error) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	for peerID, isBusy := range fs.peers {
		if isBusy {
			continue
		}

		fs.peers[peerID] = true
		return peerID, nil
	}

	return "", errNoValidFastSyncPeer
}

func (fs *fastSyncPeers) setIdle(peerID string) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	if _, ok := fs.peers[peerID]; !ok {
		return
	}

	fs.peers[peerID] = false
}

func (fs *fastSyncPeers) size() int {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	return len(fs.peers)
}
