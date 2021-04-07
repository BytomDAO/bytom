package chainmgr

import (
	"testing"
)

func TestAddDel(t *testing.T) {
	syncPeers := newFastSyncPeers()
	peers := make(map[string]bool)
	peers["Peer1"] = true
	peers["Peer2"] = true
	for k := range peers {
		syncPeers.add(k)
		syncPeers.add(k)
	}
	if syncPeers.size() != len(peers) {
		t.Errorf("add peer test err: got %d\nwant %d", syncPeers.size(), len(peers))
	}

	syncPeers.delete("Peer1")
	if syncPeers.size() != 1 {
		t.Errorf("add peer test err: got %d\nwant %d", syncPeers.size(), 1)
	}

	syncPeers.delete("Peer1")
	if syncPeers.size() != 1 {
		t.Errorf("add peer test err: got %d\nwant %d", syncPeers.size(), 1)
	}
}

func TestIdlePeers(t *testing.T) {
	syncPeers := newFastSyncPeers()
	peers := make(map[string]bool)
	peers["Peer1"] = true
	peers["Peer2"] = true
	for k := range peers {
		syncPeers.add(k)
		syncPeers.add(k)
	}

	idlePeers := syncPeers.selectIdlePeers()
	if len(idlePeers) != len(peers) {
		t.Errorf("selcet idle peers test err: got %d\nwant %d", len(idlePeers), len(peers))
	}

	for _, peer := range idlePeers {
		if ok := peers[peer]; !ok {
			t.Errorf("selcet idle peers test err: want peers %v got %v", peers, idlePeers)
		}
	}

	idlePeers = syncPeers.selectIdlePeers()
	if len(idlePeers) != 0 {
		t.Errorf("selcet idle peers test err: got %d\nwant %d", len(idlePeers), 0)
	}

}

func TestIdlePeer(t *testing.T) {
	syncPeers := newFastSyncPeers()
	peers := make(map[string]bool)
	peers["Peer1"] = true
	peers["Peer2"] = true
	for k := range peers {
		syncPeers.add(k)
		syncPeers.add(k)
	}
	idlePeer, err := syncPeers.selectIdlePeer()
	if err != nil {
		t.Errorf("selcet idle peers test err: got %v\nwant %v", err, nil)
	}

	if ok := peers[idlePeer]; !ok {
		t.Error("selcet idle peers test err.")
	}
	idlePeer, err = syncPeers.selectIdlePeer()
	if err != nil {
		t.Errorf("selcet idle peers test err: got %v\nwant %v", err, nil)
	}

	if ok := peers[idlePeer]; !ok {
		t.Error("selcet idle peers test err.")
	}
	idlePeer, err = syncPeers.selectIdlePeer()
	if err != errNoValidFastSyncPeer {
		t.Errorf("selcet idle peers test err: got %v\nwant %v", err, errNoValidFastSyncPeer)
	}
}

func TestSetIdle(t *testing.T) {
	syncPeers := newFastSyncPeers()
	peers := make(map[string]bool)
	peers["Peer2"] = true
	for k := range peers {
		syncPeers.add(k)
	}
	if syncPeers.size() != len(peers) {
		t.Errorf("add peer test err: got %d\nwant %d", syncPeers.size(), len(peers))
	}
	idlePeers := syncPeers.selectIdlePeers()
	if len(idlePeers) != len(peers) {
		t.Errorf("selcet idle peers test err: got %d\nwant %d", len(idlePeers), len(peers))
	}

	syncPeers.setIdle("Peer1")
	idlePeers = syncPeers.selectIdlePeers()
	if len(idlePeers) != 0 {
		t.Errorf("selcet idle peers test err: got %d\nwant %d", len(idlePeers), 0)
	}

	syncPeers.setIdle("Peer2")
	idlePeers = syncPeers.selectIdlePeers()
	if len(idlePeers) != len(peers) {
		t.Errorf("selcet idle peers test err: got %d\nwant %d", len(idlePeers), len(peers))
	}
}
