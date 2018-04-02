package netsync

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	cfg "github.com/bytom/config"
	"github.com/bytom/netsync/fetcher"
	"github.com/bytom/p2p"
	core "github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/version"
)

type SyncManager struct {
	networkId uint64
	sw        *p2p.Switch
	addrBook  *p2p.AddrBook // known peers

	privKey     crypto.PrivKeyEd25519 // local node's p2p key
	chain       *core.Chain
	txPool      *core.TxPool
	fetcher     *fetcher.Fetcher
	blockKeeper *blockKeeper
	peers       *peerSet

	newBlockCh    chan *bc.Hash
	newPeerCh     chan struct{}
	quitSync      chan struct{}
	config        *cfg.Config
	synchronising int32
}

func NewSyncManager(config *cfg.Config, chain *core.Chain, txPool *core.TxPool, newBlockCh chan *bc.Hash) (*SyncManager, error) {
	// Create the protocol manager with the base fields
	manager := &SyncManager{
		txPool:     txPool,
		chain:      chain,
		privKey:    crypto.GenPrivKeyEd25519(),
		config:     config,
		quitSync:   make(chan struct{}),
		newBlockCh: newBlockCh,
		peers:      newPeerSet(),
	}

	heighter := func() uint64 {
		return chain.Height()
	}

	inserter := func(block *types.Block) (bool, error) {
		return manager.chain.ProcessBlock(block)
	}

	manager.fetcher = fetcher.New(chain.GetBlockByHash, manager.BroadcastMinedBlock, heighter, inserter, manager.removePeer)

	trustHistoryDB := dbm.NewDB("trusthistory", config.DBBackend, config.DBDir())

	manager.sw = p2p.NewSwitch(config.P2P, trustHistoryDB)

	manager.blockKeeper = newBlockKeeper(manager.chain, manager.sw, manager.peers)

	protocolReactor := NewProtocolReactor(chain, txPool, manager.sw, manager.blockKeeper, manager.fetcher, manager.peers)
	manager.sw.AddReactor("PROTOCOL", protocolReactor)
	manager.newPeerCh = protocolReactor.GetNewPeerChan()

	// Optionally, start the pex reactor
	//var addrBook *p2p.AddrBook
	if config.P2P.PexReactor {
		manager.addrBook = p2p.NewAddrBook(config.P2P.AddrBookFile(), config.P2P.AddrBookStrict)
		pexReactor := p2p.NewPEXReactor(manager.addrBook)
		manager.sw.AddReactor("PEX", pexReactor)
	}

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

func (self *SyncManager) makeNodeInfo() *p2p.NodeInfo {
	nodeInfo := &p2p.NodeInfo{
		PubKey:  self.privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker: self.config.Moniker,
		Network: "bytom",
		Version: version.Version,
		Other: []string{
			cmn.Fmt("wire_version=%v", wire.Version),
			cmn.Fmt("p2p_version=%v", p2p.Version),
		},
	}

	if !self.sw.IsListening() {
		return nodeInfo
	}

	p2pListener := self.sw.Listeners()[0]
	p2pHost := p2pListener.ExternalAddress().IP.String()
	p2pPort := p2pListener.ExternalAddress().Port

	// We assume that the rpcListener has the same ExternalAddress.
	// This is probably true because both P2P and RPC listeners use UPnP,
	// except of course if the rpc is only bound to localhost
	nodeInfo.ListenAddr = cmn.Fmt("%v:%v", p2pHost, p2pPort)
	return nodeInfo
}

func (self *SyncManager) netStart() error {
	// Create & add listener
	p, address := protocolAndAddress(self.config.P2P.ListenAddress)

	l := p2p.NewDefaultListener(p, address, self.config.P2P.SkipUPNP, nil)

	self.sw.AddListener(l)

	// Start the switch
	self.sw.SetNodeInfo(self.makeNodeInfo())
	self.sw.SetNodePrivKey(self.privKey)
	_, err := self.sw.Start()
	if err != nil {
		return err
	}

	// If seeds exist, add them to the address book and dial out
	if self.config.P2P.Seeds != "" {
		// dial out
		seeds := strings.Split(self.config.P2P.Seeds, ",")
		if err := self.DialSeeds(seeds); err != nil {
			return err
		}
	}

	return nil
}

func (self *SyncManager) Start() {
	self.netStart()
	// broadcast transactions
	go self.txBroadcastLoop()

	// broadcast mined blocks
	go self.minedBroadcastLoop()

	// start sync handlers
	go self.syncer()

	//TODO:
	// go self.txsyncLoop()
}

func (self *SyncManager) Stop() {
	close(self.quitSync)
	self.sw.Stop()
}

func (self *SyncManager) txBroadcastLoop() {
	newTxCh := self.txPool.GetNewTxCh()
	for {
		select {
		case newTx := <-newTxCh:
			self.BroadcastTx(newTx)

		case <-self.quitSync:
			return
		}
	}
}

func (self *SyncManager) minedBroadcastLoop() {
	for {
		select {
		case blockHash := <-self.newBlockCh:
			block, err := self.chain.GetBlockByHash(blockHash)
			if err != nil {
				log.Errorf("Failed on mined broadcast loop get block %v", err)
				return
			}
			self.BroadcastMinedBlock(block)
		case <-self.quitSync:
			return
		}
	}
}

// BroadcastTransaction broadcats `BlockStore` transaction.
func (self *SyncManager) BroadcastTx(tx *types.Tx) {
	if err := self.blockKeeper.BroadcastTx(tx); err != nil {
		log.Errorf("SyncManager: failed on broadcast tx: %v", err)
	}
}

// BroadcastBlock will  propagate a block to it's peers.
func (self *SyncManager) BroadcastMinedBlock(block *types.Block) {
	if err := self.blockKeeper.BroadcastMinedBlock(block); err != nil {
		log.Errorf("SyncManager: failed on broadcast mined block: %v", err)
	}
}

func (self *SyncManager) NodeInfo() *p2p.NodeInfo {
	return self.sw.NodeInfo()
}

func (self *SyncManager) BlockKeeper() *blockKeeper {
	return self.blockKeeper
}

func (self *SyncManager) Peers() *peerSet {
	return self.peers
}

func (self *SyncManager) DialSeeds(seeds []string) error {
	return self.sw.DialSeeds(self.addrBook, seeds)
}

func (self *SyncManager) Switch() *p2p.Switch {
	return self.sw
}

func (self *SyncManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peers := self.sw.Peers()
	if peers == nil {
		return
	}

	peer := peers.Get(id)
	if peer == nil {
		return
	}

	peers.Remove(peer)
	log.Debug("Removing bytom peer", "peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	//pm.downloader.UnregisterPeer(id)
	//if err := pm.peers.Unregister(id); err != nil {
	//	log.Error("Peer removal failed", "peer", id, "err", err)
	//}
	// Hard disconnect at the networking layer
	//TODO
	if peer != nil {
		//peer.Peer.Disconnect(p2p.DiscUselessPeer)
		peer.CloseConn()
	}
}
