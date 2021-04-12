package netsync

import (
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/netsync/chainmgr"
	"github.com/bytom/bytom/netsync/consensusmgr"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/protocol"
)

const (
	logModule = "netsync"
)

var (
	errVaultModeDialPeer = errors.New("can't dial peer in vault mode")
)

// ChainMgr is the interface for p2p chain message sync manager.
type ChainMgr interface {
	Start() error
	IsCaughtUp() bool
	Stop()
}

// ConsensusMgr is the interface for consensus message sync manager.
type ConsensusMgr interface {
	Start() error
	Stop()
}

// Switch is the interface for p2p switch.
type Switch interface {
	Start() error
	Stop() error
	IsListening() bool
	DialPeerWithAddress(addr *p2p.NetAddress) error
	Peers() *p2p.PeerSet
}

//SyncManager Sync Manager is responsible for the business layer information synchronization
type SyncManager struct {
	config       *config.Config
	sw           Switch
	chainMgr     ChainMgr
	consensusMgr ConsensusMgr
	peers        *peers.PeerSet
}

// NewSyncManager create sync manager and set switch.
func NewSyncManager(config *config.Config, chain *protocol.Chain, txPool *protocol.TxPool, dispatcher *event.Dispatcher, fastSyncDB dbm.DB) (*SyncManager, error) {
	sw, err := p2p.NewSwitch(config)
	if err != nil {
		return nil, err
	}
	peers := peers.NewPeerSet(sw)

	chainManger, err := chainmgr.NewManager(config, sw, chain, txPool, dispatcher, peers, fastSyncDB)
	if err != nil {
		return nil, err
	}
	consensusMgr := consensusmgr.NewManager(sw, chain, peers, dispatcher)
	return &SyncManager{
		config:       config,
		sw:           sw,
		chainMgr:     chainManger,
		consensusMgr: consensusMgr,
		peers:        peers,
	}, nil
}

// Start message sync manager service.
func (sm *SyncManager) Start() error {
	if err := sm.sw.Start(); err != nil {
		logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("failed start switch")
		return err
	}

	if err := sm.chainMgr.Start(); err != nil {
		return err
	}

	return sm.consensusMgr.Start()
}

// Stop message sync manager service.
func (sm *SyncManager) Stop() {
	sm.chainMgr.Stop()
	sm.consensusMgr.Stop()
	if !sm.config.VaultMode {
		sm.sw.Stop()
	}

}

// IsListening check if the bytomd service port is open?
func (sm *SyncManager) IsListening() bool {
	if sm.config.VaultMode {
		return false
	}
	return sm.sw.IsListening()

}

//IsCaughtUp check wheather the peer finish the sync
func (sm *SyncManager) IsCaughtUp() bool {
	return sm.chainMgr.IsCaughtUp()
}

// PeerCount count the number of connected peers.
func (sm *SyncManager) PeerCount() int {
	if sm.config.VaultMode {
		return 0
	}
	return len(sm.sw.Peers().List())
}

// GetNetwork get the type of network.
func (sm *SyncManager) GetNetwork() string {
	return sm.config.ChainID
}

// BestPeer fine the peer with the highest height from the connected peers.
func (sm *SyncManager) BestPeer() *peers.PeerInfo {
	bestPeer := sm.peers.BestPeer(consensus.SFFullNode)
	if bestPeer != nil {
		return bestPeer.GetPeerInfo()
	}
	return nil
}

// DialPeerWithAddress dial the peer and establish a connection.
func (sm *SyncManager) DialPeerWithAddress(addr *p2p.NetAddress) error {
	if sm.config.VaultMode {
		return errVaultModeDialPeer
	}

	return sm.sw.DialPeerWithAddress(addr)
}

//GetPeerInfos return peer info of all connected peers.
func (sm *SyncManager) GetPeerInfos() []*peers.PeerInfo {
	return sm.peers.GetPeerInfos()
}

//StopPeer try to stop peer by given ID
func (sm *SyncManager) StopPeer(peerID string) error {
	if peer := sm.peers.GetPeer(peerID); peer == nil {
		return errors.New("peerId not exist")
	}
	sm.peers.RemovePeer(peerID)
	return nil
}
