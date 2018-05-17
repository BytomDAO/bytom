package netsync

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	cfg "github.com/bytom/config"
	"github.com/bytom/p2p"
	core "github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/version"
)

//SyncManager Sync Manager is responsible for the business layer information synchronization
type SyncManager struct {
	networkID uint64
	sw        *p2p.Switch

	privKey     crypto.PrivKeyEd25519 // local node's p2p key
	chain       *core.Chain
	txPool      *core.TxPool
	fetcher     *Fetcher
	blockKeeper *blockKeeper
	peers       *peerSet

	newBlockCh    chan *bc.Hash
	newPeerCh     chan struct{}
	txSyncCh      chan *txsync
	dropPeerCh    chan *string
	quitSync      chan struct{}
	config        *cfg.Config
	synchronising int32
}

//NewSyncManager create a sync manager
func NewSyncManager(config *cfg.Config, chain *core.Chain, txPool *core.TxPool, newBlockCh chan *bc.Hash) (*SyncManager, error) {
	// Create the protocol manager with the base fields
	manager := &SyncManager{
		txPool:     txPool,
		chain:      chain,
		privKey:    crypto.GenPrivKeyEd25519(),
		config:     config,
		quitSync:   make(chan struct{}),
		newBlockCh: newBlockCh,
		newPeerCh:  make(chan struct{}),
		txSyncCh:   make(chan *txsync),
		dropPeerCh: make(chan *string, maxQuitReq),
		peers:      newPeerSet(),
	}

	trustHistoryDB := dbm.NewDB("trusthistory", config.DBBackend, config.DBDir())
	manager.sw = p2p.NewSwitch(config.P2P, trustHistoryDB)

	manager.blockKeeper = newBlockKeeper(manager.chain, manager.sw, manager.peers, manager.dropPeerCh)
	manager.fetcher = NewFetcher(chain, manager.sw, manager.peers)

	protocolReactor := NewProtocolReactor(chain, txPool, manager.sw, manager.blockKeeper, manager.fetcher, manager.peers, manager.newPeerCh, manager.txSyncCh, manager.dropPeerCh)
	manager.sw.AddReactor("PROTOCOL", protocolReactor)

	// Create & add listener
	var listenerStatus bool
	var l p2p.Listener
	if !config.VaultMode {
		p, address := protocolAndAddress(manager.config.P2P.ListenAddress)
		l, listenerStatus = p2p.NewDefaultListener(p, address, manager.config.P2P.SkipUPNP, nil)
		manager.sw.AddListener(l)
	}
	manager.sw.SetNodeInfo(manager.makeNodeInfo(listenerStatus))
	manager.sw.SetNodePrivKey(manager.privKey)

	return manager, nil
}

// Defaults to tcp
func protocolAndAddress(listenAddr string) (string, string) {
	p, address := "tcp", listenAddr
	parts := strings.SplitN(address, "://", 2)
	if len(parts) == 2 {
		p, address = parts[0], parts[1]
	}
	return p, address
}

func (sm *SyncManager) makeNodeInfo(listenerStatus bool) *p2p.NodeInfo {
	nodeInfo := &p2p.NodeInfo{
		PubKey:  sm.privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker: sm.config.Moniker,
		Network: sm.config.ChainID,
		Version: version.Version,
		Other: []string{
			cmn.Fmt("wire_version=%v", wire.Version),
			cmn.Fmt("p2p_version=%v", p2p.Version),
		},
	}

	if !sm.sw.IsListening() {
		return nodeInfo
	}

	p2pListener := sm.sw.Listeners()[0]

	// We assume that the rpcListener has the same ExternalAddress.
	// This is probably true because both P2P and RPC listeners use UPnP,
	// except of course if the rpc is only bound to localhost
	if listenerStatus {
		nodeInfo.ListenAddr = cmn.Fmt("%v:%v", p2pListener.ExternalAddress().IP.String(), p2pListener.ExternalAddress().Port)
	} else {
		nodeInfo.ListenAddr = cmn.Fmt("%v:%v", p2pListener.InternalAddress().IP.String(), p2pListener.InternalAddress().Port)
	}
	return nodeInfo
}

func (sm *SyncManager) netStart() error {
	// Start the switch
	_, err := sm.sw.Start()
	if err != nil {
		return err
	}

	// If seeds exist, add them to the address book and dial out
	if sm.config.P2P.Seeds != "" {
		// dial out
		seeds := strings.Split(sm.config.P2P.Seeds, ",")
		if err := sm.DialSeeds(seeds); err != nil {
			return err
		}
	}
	return nil
}

//Start start sync manager service
func (sm *SyncManager) Start() {
	go sm.netStart()
	// broadcast transactions
	go sm.txBroadcastLoop()

	// broadcast mined blocks
	go sm.minedBroadcastLoop()

	// start sync handlers
	go sm.syncer()

	go sm.txsyncLoop()
}

//Stop stop sync manager
func (sm *SyncManager) Stop() {
	close(sm.quitSync)
	sm.sw.Stop()
}

func (sm *SyncManager) txBroadcastLoop() {
	newTxCh := sm.txPool.GetNewTxCh()
	for {
		select {
		case newTx := <-newTxCh:
			peers, err := sm.peers.BroadcastTx(newTx)
			if err != nil {
				log.Errorf("Broadcast new tx error. %v", err)
				return
			}
			for _, smPeer := range peers {
				if smPeer == nil {
					continue
				}
				swPeer := smPeer.getPeer()
				log.Info("Tx broadcast error. Stop Peer.")
				sm.sw.StopPeerGracefully(swPeer)
			}
		case <-sm.quitSync:
			return
		}
	}
}

func (sm *SyncManager) minedBroadcastLoop() {
	for {
		select {
		case blockHash := <-sm.newBlockCh:
			block, err := sm.chain.GetBlockByHash(blockHash)
			if err != nil {
				log.Errorf("Failed on mined broadcast loop get block %v", err)
				return
			}
			peers, err := sm.peers.BroadcastMinedBlock(block)
			if err != nil {
				log.Errorf("Broadcast mine block error. %v", err)
				return
			}
			for _, smPeer := range peers {
				if smPeer == nil {
					continue
				}
				swPeer := smPeer.getPeer()
				log.Info("New mined block broadcast error. Stop Peer.")
				sm.sw.StopPeerGracefully(swPeer)
			}
		case <-sm.quitSync:
			return
		}
	}
}

//NodeInfo get P2P peer node info
func (sm *SyncManager) NodeInfo() *p2p.NodeInfo {
	return sm.sw.NodeInfo()
}

//BlockKeeper get block keeper
func (sm *SyncManager) BlockKeeper() *blockKeeper {
	return sm.blockKeeper
}

//Peers get sync manager peer set
func (sm *SyncManager) Peers() *peerSet {
	return sm.peers
}

//DialSeeds dial seed peers
func (sm *SyncManager) DialSeeds(seeds []string) error {
	return sm.sw.DialSeeds(seeds)
}

//Switch get sync manager switch
func (sm *SyncManager) Switch() *p2p.Switch {
	return sm.sw
}
