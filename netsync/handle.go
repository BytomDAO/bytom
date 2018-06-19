package netsync

import (
	"encoding/hex"
	"net"
	"path"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/p2p"
	"github.com/bytom/p2p/discover"
	"github.com/bytom/p2p/pex"
	core "github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
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

	newTxCh       chan *types.Tx
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
	sw := p2p.NewSwitch(config)
	peers := newPeerSet()
	dropPeerCh := make(chan *string, maxQuitReq)
	manager := &SyncManager{
		sw:          sw,
		txPool:      txPool,
		chain:       chain,
		privKey:     crypto.GenPrivKeyEd25519(),
		fetcher:     NewFetcher(chain, sw, peers),
		blockKeeper: newBlockKeeper(chain, sw, peers, dropPeerCh),
		peers:       peers,
		newTxCh:     make(chan *types.Tx, maxTxChanSize),
		newBlockCh:  newBlockCh,
		newPeerCh:   make(chan struct{}),
		txSyncCh:    make(chan *txsync),
		dropPeerCh:  dropPeerCh,
		quitSync:    make(chan struct{}),
		config:      config,
	}

	protocolReactor := NewProtocolReactor(chain, txPool, manager.sw, manager.blockKeeper, manager.fetcher, manager.peers, manager.newPeerCh, manager.txSyncCh, manager.dropPeerCh)
	manager.sw.AddReactor("PROTOCOL", protocolReactor)

	// Create & add listener
	var listenerStatus bool
	var l p2p.Listener
	if !config.VaultMode {
		p, address := protocolAndAddress(manager.config.P2P.ListenAddress)
		l, listenerStatus = p2p.NewDefaultListener(p, address, manager.config.P2P.SkipUPNP)
		manager.sw.AddListener(l)

		discv, err := initDiscover(config, &manager.privKey, l.ExternalAddress().Port)
		if err != nil {
			return nil, err
		}

		pexReactor := pex.NewPEXReactor(discv)
		manager.sw.AddReactor("PEX", pexReactor)
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
		Other:   []string{strconv.FormatUint(uint64(consensus.DefaultServices), 10)},
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

//Start start sync manager service
func (sm *SyncManager) Start() {
	if _, err := sm.sw.Start(); err != nil {
		cmn.Exit(cmn.Fmt("fail on start SyncManager: %v", err))
	}
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

func initDiscover(config *cfg.Config, priv *crypto.PrivKeyEd25519, port uint16) (*discover.Network, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("0.0.0.0", strconv.FormatUint(uint64(port), 10)))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	ntab, err := discover.ListenUDP(priv, conn, realaddr, path.Join(config.DBDir(), "discover.db"), nil)
	if err != nil {
		return nil, err
	}

	// add the seeds node to the discover table
	if config.P2P.Seeds == "" {
		return ntab, nil
	}
	nodes := []*discover.Node{}
	for _, seed := range strings.Split(config.P2P.Seeds, ",") {
		url := "enode://" + hex.EncodeToString(crypto.Sha256([]byte(seed))) + "@" + seed
		nodes = append(nodes, discover.MustParseNode(url))
	}
	if err = ntab.SetFallbackNodes(nodes); err != nil {
		return nil, err
	}
	return ntab, nil
}

func (sm *SyncManager) txBroadcastLoop() {
	for {
		select {
		case newTx := <-sm.newTxCh:
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

//Switch get sync manager switch
func (sm *SyncManager) Switch() *p2p.Switch {
	return sm.sw
}

// GetNewTxCh return a unconfirmed transaction feed channel
func (sm *SyncManager) GetNewTxCh() chan *types.Tx {
	return sm.newTxCh
}
