package node

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/accesstoken"
	"github.com/bytom/account"
	"github.com/bytom/api"
	"github.com/bytom/asset"
	bc "github.com/bytom/blockchain"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txfeed"
	cfg "github.com/bytom/config"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/env"
	"github.com/bytom/p2p"
	"github.com/bytom/protocol"
	"github.com/bytom/types"
	"github.com/bytom/util/browser"
	"github.com/bytom/version"
	w "github.com/bytom/wallet"
)

const (
	webAddress               = "http://127.0.0.1:9888"
	expireReservationsPeriod = time.Second
)

type Node struct {
	cmn.BaseService

	// config
	config *cfg.Config

	// network
	privKey  crypto.PrivKeyEd25519 // local node's p2p key
	sw       *p2p.Switch           // p2p connections
	addrBook *p2p.AddrBook         // known peers

	evsw         types.EventSwitch // pub/sub for services
	bcReactor    *bc.BlockchainReactor
	wallet       *w.Wallet
	accessTokens *accesstoken.CredentialStore
	api          *api.API
	chain        *protocol.Chain
}

func NewNode(config *cfg.Config) *Node {
	ctx := context.Background()

	// Get store
	txDB := dbm.NewDB("txdb", config.DBBackend, config.DBDir())
	store := leveldb.NewStore(txDB)

	tokenDB := dbm.NewDB("accesstoken", config.DBBackend, config.DBDir())
	accessTokens := accesstoken.NewStore(tokenDB)

	privKey := crypto.GenPrivKeyEd25519()

	// Make event switch
	eventSwitch := types.NewEventSwitch()
	_, err := eventSwitch.Start()
	if err != nil {
		cmn.Exit(cmn.Fmt("Failed to start switch: %v", err))
	}

	trustHistoryDB := dbm.NewDB("trusthistory", config.DBBackend, config.DBDir())

	sw := p2p.NewSwitch(config.P2P, trustHistoryDB)

	genesisBlock := cfg.GenerateGenesisBlock()

	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(genesisBlock.Hash(), store, txPool)
	if err != nil {
		cmn.Exit(cmn.Fmt("Failed to create chain structure: %v", err))
	}

	if chain.BestBlockHash() == nil {
		if err := chain.SaveBlock(genesisBlock); err != nil {
			cmn.Exit(cmn.Fmt("Failed to save genesisBlock to store: %v", err))
		}
		if err := chain.ConnectBlock(genesisBlock); err != nil {
			cmn.Exit(cmn.Fmt("Failed to connect genesisBlock to chain: %v", err))
		}
	}

	var accounts *account.Manager = nil
	var assets *asset.Registry = nil
	var wallet *w.Wallet = nil
	var txFeed *txfeed.Tracker = nil

	txFeedDB := dbm.NewDB("txfeeds", config.DBBackend, config.DBDir())
	txFeed = txfeed.NewTracker(txFeedDB, chain)

	if err = txFeed.Prepare(ctx); err != nil {
		log.WithField("error", err).Error("start txfeed")
		return nil
	}

	hsm, err := pseudohsm.New(config.KeysDir())
	if err != nil {
		cmn.Exit(cmn.Fmt("initialize HSM failed: %v", err))
	}

	if !config.Wallet.Disable {
		walletDB := dbm.NewDB("wallet", config.DBBackend, config.DBDir())
		accounts = account.NewManager(walletDB, chain)
		assets = asset.NewRegistry(walletDB, chain)
		wallet, err = w.NewWallet(walletDB, accounts, assets, hsm, chain)
		if err != nil {
			log.WithField("error", err).Error("init NewWallet")
		}

		if err := initOrRecoverAccount(hsm, wallet); err != nil {
			log.WithField("error", err).Error("initialize or recover account")
		}

		// Clean up expired UTXO reservations periodically.
		go accounts.ExpireReservations(ctx, expireReservationsPeriod)
	}

	bcReactor := bc.NewBlockchainReactor(chain, txPool, sw, accounts, txFeed, config.Mining)

	sw.AddReactor("BLOCKCHAIN", bcReactor)

	// Optionally, start the pex reactor
	var addrBook *p2p.AddrBook
	if config.P2P.PexReactor {
		addrBook = p2p.NewAddrBook(config.P2P.AddrBookFile(), config.P2P.AddrBookStrict)
		pexReactor := p2p.NewPEXReactor(addrBook)
		sw.AddReactor("PEX", pexReactor)
	}

	// run the profile server
	profileHost := config.ProfListenAddress
	if profileHost != "" {
		// Profiling bytomd programs.see (https://blog.golang.org/profiling-go-programs)
		// go tool pprof http://profileHose/debug/pprof/heap
		go func() {
			http.ListenAndServe(profileHost, nil)
		}()
	}

	node := &Node{
		config: config,

		privKey:  privKey,
		sw:       sw,
		addrBook: addrBook,

		evsw:         eventSwitch,
		bcReactor:    bcReactor,
		accessTokens: accessTokens,
		wallet:       wallet,
		chain:        chain,
	}
	node.BaseService = *cmn.NewBaseService(nil, "Node", node)

	return node
}

func initOrRecoverAccount(hsm *pseudohsm.HSM, wallet *w.Wallet) error {
	xpubs := hsm.ListKeys()

	if len(xpubs) == 0 {
		xpub, err := hsm.XCreate("default", "123456")
		if err != nil {
			return err
		}

		wallet.AccountMgr.Create(nil, []chainkd.XPub{xpub.XPub}, 1, "default", nil)
		return nil
	}

	accounts, err := wallet.AccountMgr.ListAccounts("")
	if err != nil {
		return err
	}

	if len(accounts) > 0 {
		return nil
	}

	for i, xPub := range xpubs {
		if err := wallet.ImportAccountXpubKey(i, xPub, w.RecoveryIndex); err != nil {
			return err
		}
	}
	return nil
}

// Lanch web broser or not
func lanchWebBroser() {
	log.Info("Launching System Browser with :", webAddress)
	if err := browser.Open(webAddress); err != nil {
		log.Error(err.Error())
		return
	}
}

func (n *Node) initAndstartApiServer() {
	n.api = api.NewAPI(n.bcReactor, n.wallet, n.chain, n.config, n.accessTokens)

	listenAddr := env.String("LISTEN", n.config.ApiAddress)
	n.api.StartServer(*listenAddr)
}

func (n *Node) OnStart() error {
	// Create & add listener
	p, address := ProtocolAndAddress(n.config.P2P.ListenAddress)
	l := p2p.NewDefaultListener(p, address, n.config.P2P.SkipUPNP, nil)
	n.sw.AddListener(l)

	// Start the switch
	n.sw.SetNodeInfo(n.makeNodeInfo())
	n.sw.SetNodePrivKey(n.privKey)
	_, err := n.sw.Start()
	if err != nil {
		return err
	}

	// If seeds exist, add them to the address book and dial out
	if n.config.P2P.Seeds != "" {
		// dial out
		seeds := strings.Split(n.config.P2P.Seeds, ",")
		if err := n.DialSeeds(seeds); err != nil {
			return err
		}
	}

	n.initAndstartApiServer()
	if !n.config.Web.Closed {
		lanchWebBroser()
	}

	return nil
}

func (n *Node) OnStop() {
	n.BaseService.OnStop()

	log.Info("Stopping Node")
	// TODO: gracefully disconnect from peers.
	n.sw.Stop()

}

func (n *Node) RunForever() {
	// Sleep forever and then...
	cmn.TrapSignal(func() {
		n.Stop()
	})
}

// Add a Listener to accept inbound peer connections.
// Add listeners before starting the Node.
// The first listener is the primary listener (in NodeInfo)
func (n *Node) AddListener(l p2p.Listener) {
	n.sw.AddListener(l)
}

func (n *Node) Switch() *p2p.Switch {
	return n.sw
}

func (n *Node) EventSwitch() types.EventSwitch {
	return n.evsw
}

func (n *Node) makeNodeInfo() *p2p.NodeInfo {
	nodeInfo := &p2p.NodeInfo{
		PubKey:  n.privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker: n.config.Moniker,
		Network: "bytom",
		Version: version.Version,
		Other: []string{
			cmn.Fmt("wire_version=%v", wire.Version),
			cmn.Fmt("p2p_version=%v", p2p.Version),
		},
	}

	if !n.sw.IsListening() {
		return nodeInfo
	}

	p2pListener := n.sw.Listeners()[0]
	p2pHost := p2pListener.ExternalAddress().IP.String()
	p2pPort := p2pListener.ExternalAddress().Port
	//rpcListenAddr := n.config.RPC.ListenAddress

	// We assume that the rpcListener has the same ExternalAddress.
	// This is probably true because both P2P and RPC listeners use UPnP,
	// except of course if the rpc is only bound to localhost
	nodeInfo.ListenAddr = cmn.Fmt("%v:%v", p2pHost, p2pPort)
	//nodeInfo.Other = append(nodeInfo.Other, cmn.Fmt("rpc_addr=%v", rpcListenAddr))
	return nodeInfo
}

//------------------------------------------------------------------------------

func (n *Node) NodeInfo() *p2p.NodeInfo {
	return n.sw.NodeInfo()
}

func (n *Node) DialSeeds(seeds []string) error {
	return n.sw.DialSeeds(n.addrBook, seeds)
}

// Defaults to tcp
func ProtocolAndAddress(listenAddr string) (string, string) {
	p, address := "tcp", listenAddr
	parts := strings.SplitN(address, "://", 2)
	if len(parts) == 2 {
		p, address = parts[0], parts[1]
	}
	return p, address
}

//------------------------------------------------------------------------------
