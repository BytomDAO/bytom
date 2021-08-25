package monitor

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	vaporCfg "github.com/bytom/bytom/config"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/netsync/chainmgr"
	"github.com/bytom/bytom/netsync/consensusmgr"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/p2p/discover/dht"
	"github.com/bytom/bytom/p2p/discover/mdns"
	"github.com/bytom/bytom/test/mock"
	"github.com/bytom/bytom/toolbar/precognitive/config"
)

type monitor struct {
	cfg            *config.Config
	db             *gorm.DB
	nodeCfg        *vaporCfg.Config
	sw             *p2p.Switch
	privKey        chainkd.XPrv
	chain          *mock.Chain
	txPool         *mock.Mempool
	bestHeightSeen uint64
	peers          *peers.PeerSet
}

func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	dbPath, err := makePath()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("makePath")
	}

	nodeCfg := &vaporCfg.Config{
		BaseConfig: vaporCfg.DefaultBaseConfig(),
		P2P:        vaporCfg.DefaultP2PConfig(),
		// Federation: vaporCfg.DefaultFederationConfig(),
	}
	nodeCfg.DBPath = dbPath
	nodeCfg.ChainID = "mainnet"
	privKey, err := chainkd.NewXPrv(nil)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("NewPrivKey")
	}

	chain, txPool, err := mockChainAndPool()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("mockChainAndPool")
	}

	return &monitor{
		cfg:            cfg,
		db:             db,
		nodeCfg:        nodeCfg,
		privKey:        privKey,
		chain:          chain,
		txPool:         txPool,
		bestHeightSeen: uint64(0),
	}
}

func makePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dataPath := path.Join(usr.HomeDir, "/.bytom_precog")
	if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
		return "", err
	}

	return dataPath, nil
}

func (m *monitor) Run() {
	if err := m.makeSwitch(); err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("makeSwitch")
	}

	go m.discoveryRoutine()
	go m.connectionRoutine()
}

func (m *monitor) makeSwitch() error {
	var seeds []string
	for _, node := range m.cfg.Nodes {
		seeds = append(seeds, fmt.Sprintf("%s:%d", node.IP, node.Port))
	}
	m.nodeCfg.P2P.Seeds = strings.Join(seeds, ",")

	l, listenAddr := p2p.GetListener(m.nodeCfg.P2P)
	discv, err := dht.NewDiscover(m.nodeCfg, m.privKey, l.ExternalAddress().Port)
	if err != nil {
		return err
	}

	// no need for lanDiscv, but passing &mdns.LANDiscover{} will cause NilPointer
	lanDiscv := mdns.NewLANDiscover(mdns.NewProtocol(m.nodeCfg.ChainID), int(l.ExternalAddress().Port))
	m.sw, err = p2p.NewSwitchDetail(m.nodeCfg, discv, lanDiscv, l, m.privKey, listenAddr)
	if err != nil {
		return err
	}

	m.peers = peers.NewPeerSet(m.sw)
	return m.prepareReactors()
}

func (m *monitor) prepareReactors() error {
	dispatcher := event.NewDispatcher()
	// add ConsensusReactor for consensusChannel
	_ = consensusmgr.NewManager(m.sw, m.chain, m.peers, dispatcher)
	fastSyncDB := dbm.NewDB("fastsync", m.nodeCfg.DBBackend, m.nodeCfg.DBDir())
	// add ProtocolReactor to handle msgs
	if _, err := chainmgr.NewManager(m.nodeCfg, m.sw, m.chain, m.txPool, dispatcher, m.peers, fastSyncDB); err != nil {
		return err
	}

	for label, reactor := range m.sw.GetReactors() {
		log.WithFields(log.Fields{"label": label, "reactor": reactor}).Debug("start reactor")
		if err := reactor.Start(); err != nil {
			return err
		}
	}

	m.sw.GetSecurity().RegisterFilter(m.sw.GetNodeInfo())
	m.sw.GetSecurity().RegisterFilter(m.sw.GetPeers())
	return m.sw.GetSecurity().Start()
}
