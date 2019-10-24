package p2p

import (
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/errors"
	"github.com/bytom/event"
	"github.com/bytom/p2p/connection"
	"github.com/bytom/p2p/discover/dht"
	"github.com/bytom/p2p/discover/mdns"
	"github.com/bytom/p2p/netutil"
	"github.com/bytom/p2p/security"
	"github.com/bytom/version"
)

const (
	logModule = "p2p"

	minNumOutboundPeers = 4
	maxNumLANPeers      = 5
)

//pre-define errors for connecting fail
var (
	ErrDuplicatePeer  = errors.New("Duplicate peer")
	ErrConnectSelf    = errors.New("Connect self")
	ErrConnectSpvPeer = errors.New("Outbound connect spv peer")
)

type discv interface {
	ReadRandomNodes(buf []*dht.Node) (n int)
}

type lanDiscv interface {
	Subscribe() (*event.Subscription, error)
	Stop()
}

type Security interface {
	DoFilter(ip string, pubKey string) error
	IsBanned(ip string, level byte, reason string) bool
	RegisterFilter(filter security.Filter)
	Start() error
}

// Switch handles peer connections and exposes an API to receive incoming messages
// on `Reactors`.  Each `Reactor` is responsible for handling incoming messages of one
// or more `Channels`.  So while sending outgoing messages is typically performed on the peer,
// incoming messages are received on the reactor.
type Switch struct {
	cmn.BaseService

	Config       *cfg.Config
	peerConfig   *PeerConfig
	listeners    []Listener
	reactors     map[string]Reactor
	chDescs      []*connection.ChannelDescriptor
	reactorsByCh map[byte]Reactor
	peers        *PeerSet
	dialing      *cmn.CMap
	nodeInfo     *NodeInfo             // our node info
	nodePrivKey  crypto.PrivKeyEd25519 // our node privkey
	discv        discv
	lanDiscv     lanDiscv
	security     Security
}

// NewSwitch create a new Switch and set discover.
func NewSwitch(config *cfg.Config) (*Switch, error) {
	var err error
	var l Listener
	var listenAddr string
	var discv *dht.Network
	var lanDiscv *mdns.LANDiscover

	config.P2P.PrivateKey, err = config.NodeKey()
	if err != nil {
		return nil, err
	}

	bytes, err := hex.DecodeString(config.P2P.PrivateKey)
	if err != nil {
		return nil, err
	}

	var newKey [64]byte
	copy(newKey[:], bytes)
	privKey := crypto.PrivKeyEd25519(newKey)
	if !config.VaultMode {
		// Create listener
		l, listenAddr = GetListener(config.P2P)
		discv, err = dht.NewDiscover(config, ed25519.PrivateKey(bytes), l.ExternalAddress().Port)
		if err != nil {
			return nil, err
		}
		if config.P2P.LANDiscover {
			lanDiscv = mdns.NewLANDiscover(mdns.NewProtocol(config.ChainID), int(l.ExternalAddress().Port))
		}
	}

	return newSwitch(config, discv, lanDiscv, l, privKey, listenAddr)
}

// newSwitch creates a new Switch with the given config.
func newSwitch(config *cfg.Config, discv discv, lanDiscv lanDiscv, l Listener, priv crypto.PrivKeyEd25519, listenAddr string) (*Switch, error) {
	sw := &Switch{
		Config:       config,
		peerConfig:   DefaultPeerConfig(config.P2P),
		reactors:     make(map[string]Reactor),
		chDescs:      make([]*connection.ChannelDescriptor, 0),
		reactorsByCh: make(map[byte]Reactor),
		peers:        NewPeerSet(),
		dialing:      cmn.NewCMap(),
		nodePrivKey:  priv,
		discv:        discv,
		lanDiscv:     lanDiscv,
		nodeInfo:     NewNodeInfo(config, priv.PubKey().Unwrap().(crypto.PubKeyEd25519), listenAddr),
		security:     security.NewSecurity(config),
	}

	sw.AddListener(l)
	sw.BaseService = *cmn.NewBaseService(nil, "P2P Switch", sw)
	return sw, nil
}

// OnStart implements BaseService. It starts all the reactors, peers, and listeners.
func (sw *Switch) OnStart() error {
	for _, reactor := range sw.reactors {
		if _, err := reactor.Start(); err != nil {
			return err
		}
	}

	sw.security.RegisterFilter(sw.nodeInfo)
	sw.security.RegisterFilter(sw.peers)
	if err := sw.security.Start(); err != nil {
		return err
	}

	for _, listener := range sw.listeners {
		go sw.listenerRoutine(listener)
	}
	go sw.ensureOutboundPeersRoutine()
	go sw.connectLANPeersRoutine()

	return nil
}

// OnStop implements BaseService. It stops all listeners, peers, and reactors.
func (sw *Switch) OnStop() {
	if sw.Config.P2P.LANDiscover {
		sw.lanDiscv.Stop()
	}

	for _, listener := range sw.listeners {
		listener.Stop()
	}
	sw.listeners = nil

	for _, peer := range sw.peers.List() {
		peer.Stop()
		sw.peers.Remove(peer)
	}

	for _, reactor := range sw.reactors {
		reactor.Stop()
	}
}

// AddPeer performs the P2P handshake with a peer
// that already has a SecretConnection. If all goes well,
// it starts the peer and adds it to the switch.
// NOTE: This performs a blocking handshake before the peer is added.
// CONTRACT: If error is returned, peer is nil, and conn is immediately closed.
func (sw *Switch) AddPeer(pc *peerConn, isLAN bool) error {
	peerNodeInfo, err := pc.HandshakeTimeout(sw.nodeInfo, sw.peerConfig.HandshakeTimeout)
	if err != nil {
		return err
	}

	if err := version.Status.CheckUpdate(sw.nodeInfo.Version, peerNodeInfo.Version, peerNodeInfo.RemoteAddr); err != nil {
		return err
	}
	if err := sw.nodeInfo.CompatibleWith(peerNodeInfo); err != nil {
		return err
	}

	peer := newPeer(pc, peerNodeInfo, sw.reactorsByCh, sw.chDescs, sw.StopPeerForError, isLAN)
	if err := sw.security.DoFilter(peer.RemoteAddrHost(), peer.PubKey().String()); err != nil {
		return err
	}

	if pc.outbound && !peer.ServiceFlag().IsEnable(consensus.SFFullNode) {
		return ErrConnectSpvPeer
	}

	// Start peer
	if sw.IsRunning() {
		if err := sw.startInitPeer(peer); err != nil {
			return err
		}
	}

	return sw.peers.Add(peer)
}

// AddReactor adds the given reactor to the switch.
// NOTE: Not goroutine safe.
func (sw *Switch) AddReactor(name string, reactor Reactor) Reactor {
	// Validate the reactor.
	// No two reactors can share the same channel.
	for _, chDesc := range reactor.GetChannels() {
		chID := chDesc.ID
		if sw.reactorsByCh[chID] != nil {
			cmn.PanicSanity(fmt.Sprintf("Channel %X has multiple reactors %v & %v", chID, sw.reactorsByCh[chID], reactor))
		}
		sw.chDescs = append(sw.chDescs, chDesc)
		sw.reactorsByCh[chID] = reactor
	}
	sw.reactors[name] = reactor
	reactor.SetSwitch(sw)
	return reactor
}

// AddListener adds the given listener to the switch for listening to incoming peer connections.
// NOTE: Not goroutine safe.
func (sw *Switch) AddListener(l Listener) {
	sw.listeners = append(sw.listeners, l)
}

//DialPeerWithAddress dial node from net address
func (sw *Switch) DialPeerWithAddress(addr *NetAddress) error {
	log.WithFields(log.Fields{"module": logModule, "address": addr}).Debug("Dialing peer")
	sw.dialing.Set(addr.IP.String(), addr)
	defer sw.dialing.Delete(addr.IP.String())
	if err := sw.security.DoFilter(addr.IP.String(), ""); err != nil {
		return err
	}

	pc, err := newOutboundPeerConn(addr, sw.nodePrivKey, sw.peerConfig)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "address": addr, " err": err}).Warn("DialPeer fail on newOutboundPeerConn")
		return err
	}

	if err = sw.AddPeer(pc, addr.isLAN); err != nil {
		log.WithFields(log.Fields{"module": logModule, "address": addr, " err": err}).Warn("DialPeer fail on switch AddPeer")
		pc.CloseConn()
		return err
	}
	log.WithFields(log.Fields{"module": logModule, "address": addr, "peer num": sw.peers.Size()}).Debug("DialPeer added peer")
	return nil
}

func (sw *Switch) IsBanned(ip string, level byte, reason string) bool {
	return sw.security.IsBanned(ip, level, reason)
}

//IsDialing prevent duplicate dialing
func (sw *Switch) IsDialing(addr *NetAddress) bool {
	return sw.dialing.Has(addr.IP.String())
}

// IsListening returns true if the switch has at least one listener.
// NOTE: Not goroutine safe.
func (sw *Switch) IsListening() bool {
	return len(sw.listeners) > 0
}

// Listeners returns the list of listeners the switch listens on.
// NOTE: Not goroutine safe.
func (sw *Switch) Listeners() []Listener {
	return sw.listeners
}

// NumPeers Returns the count of outbound/inbound and outbound-dialing peers.
func (sw *Switch) NumPeers() (lan, outbound, inbound, dialing int) {
	peers := sw.peers.List()
	for _, peer := range peers {
		if peer.outbound && !peer.isLAN {
			outbound++
		} else {
			inbound++
		}
		if peer.isLAN {
			lan++
		}
	}
	dialing = sw.dialing.Size()
	return
}

// NodeInfo returns the switch's NodeInfo.
// NOTE: Not goroutine safe.
func (sw *Switch) NodeInfo() *NodeInfo {
	return sw.nodeInfo
}

//Peers return switch peerset
func (sw *Switch) Peers() *PeerSet {
	return sw.peers
}

// StopPeerForError disconnects from a peer due to external error.
func (sw *Switch) StopPeerForError(peer *Peer, reason interface{}) {
	log.WithFields(log.Fields{"module": logModule, "peer": peer, " err": reason}).Debug("stopping peer for error")
	sw.stopAndRemovePeer(peer, reason)
}

// StopPeerGracefully disconnect from a peer gracefully.
func (sw *Switch) StopPeerGracefully(peerID string) {
	if peer := sw.peers.Get(peerID); peer != nil {
		sw.stopAndRemovePeer(peer, nil)
	}
}

func (sw *Switch) addPeerWithConnection(conn net.Conn) error {
	peerConn, err := newInboundPeerConn(conn, sw.nodePrivKey, sw.Config.P2P)
	if err != nil {
		if err := conn.Close(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "remote peer:": conn.RemoteAddr().String(), " err:": err}).Warn("closes connection err")
		}
		return err
	}

	if err = sw.AddPeer(peerConn, false); err != nil {
		if err := conn.Close(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "remote peer:": conn.RemoteAddr().String(), " err:": err}).Warn("closes connection err")
		}
		return err
	}

	log.WithFields(log.Fields{"module": logModule, "address": conn.RemoteAddr().String(), "peer num": sw.peers.Size()}).Debug("add inbound peer")
	return nil
}

func (sw *Switch) connectLANPeers(lanPeer mdns.LANPeerEvent) {
	lanPeers, _, _, numDialing := sw.NumPeers()
	numToDial := maxNumLANPeers - lanPeers
	log.WithFields(log.Fields{"module": logModule, "numDialing": numDialing, "numToDial": numToDial}).Debug("connect LAN peers")
	if numToDial <= 0 {
		return
	}
	addresses := make([]*NetAddress, 0)
	for i := 0; i < len(lanPeer.IP); i++ {
		addresses = append(addresses, NewLANNetAddressIPPort(lanPeer.IP[i], uint16(lanPeer.Port)))
	}
	sw.dialPeers(addresses)
}

func (sw *Switch) connectLANPeersRoutine() {
	if !sw.Config.P2P.LANDiscover {
		return
	}

	lanPeerEventSub, err := sw.lanDiscv.Subscribe()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("subscribe LAN Peer Event error")
		return
	}

	for {
		select {
		case obj, ok := <-lanPeerEventSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("LAN peer event subscription channel closed")
				return
			}
			LANPeer, ok := obj.Data.(mdns.LANPeerEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}
			sw.connectLANPeers(LANPeer)
		case <-sw.Quit:
			return
		}
	}
}

func (sw *Switch) listenerRoutine(l Listener) {
	for {
		inConn, ok := <-l.Connections()
		if !ok {
			break
		}

		// disconnect if we alrady have MaxNumPeers
		if sw.peers.Size() >= sw.Config.P2P.MaxNumPeers {
			if err := inConn.Close(); err != nil {
				log.WithFields(log.Fields{"module": logModule, "remote peer:": inConn.RemoteAddr().String(), " err:": err}).Warn("closes connection err")
			}
			log.Info("Ignoring inbound connection: already have enough peers.")
			continue
		}

		// New inbound connection!
		if err := sw.addPeerWithConnection(inConn); err != nil {
			log.Info("Ignoring inbound connection: error while adding peer.", " address:", inConn.RemoteAddr().String(), " error:", err)
			continue
		}
	}
}

func (sw *Switch) dialPeerWorker(a *NetAddress, wg *sync.WaitGroup) {
	if err := sw.DialPeerWithAddress(a); err != nil {
		log.WithFields(log.Fields{"module": logModule, "addr": a, "err": err}).Warn("dialPeerWorker fail on dial peer")
	}
	wg.Done()
}

func (sw *Switch) dialPeers(addresses []*NetAddress) {
	connectedPeers := make(map[string]struct{})
	for _, peer := range sw.Peers().List() {
		connectedPeers[peer.RemoteAddrHost()] = struct{}{}
	}

	var wg sync.WaitGroup
	for _, address := range addresses {
		if sw.NodeInfo().ListenAddr == address.String() {
			continue
		}
		if dialling := sw.IsDialing(address); dialling {
			continue
		}
		if _, ok := connectedPeers[address.IP.String()]; ok {
			continue
		}

		wg.Add(1)
		go sw.dialPeerWorker(address, &wg)
	}
	wg.Wait()
}

func (sw *Switch) ensureKeepConnectPeers() {
	keepDials := netutil.CheckAndSplitAddresses(sw.Config.P2P.KeepDial)
	addresses := make([]*NetAddress, 0)
	for _, keepDial := range keepDials {
		address, err := NewNetAddressString(keepDial)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err, "address": keepDial}).Warn("parse address to NetAddress")
			continue
		}
		addresses = append(addresses, address)
	}

	sw.dialPeers(addresses)
}

func (sw *Switch) ensureOutboundPeers() {
	lanPeers, numOutPeers, _, numDialing := sw.NumPeers()
	numToDial := minNumOutboundPeers - (numOutPeers + numDialing)
	log.WithFields(log.Fields{"module": logModule, "numOutPeers": numOutPeers, "LANPeers": lanPeers, "numDialing": numDialing, "numToDial": numToDial}).Debug("ensure peers")
	if numToDial <= 0 {
		return
	}

	nodes := make([]*dht.Node, numToDial)
	n := sw.discv.ReadRandomNodes(nodes)
	addresses := make([]*NetAddress, 0)
	for i := 0; i < n; i++ {
		address := NewNetAddressIPPort(nodes[i].IP, nodes[i].TCP)
		addresses = append(addresses, address)
	}
	sw.dialPeers(addresses)
}

func (sw *Switch) ensureOutboundPeersRoutine() {
	sw.ensureKeepConnectPeers()
	sw.ensureOutboundPeers()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sw.ensureKeepConnectPeers()
			sw.ensureOutboundPeers()
		case <-sw.Quit:
			return
		}
	}
}

func (sw *Switch) startInitPeer(peer *Peer) error {
	// spawn send/recv routines
	if _, err := peer.Start(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "remote peer:": peer.RemoteAddr, " err:": err}).Error("init peer err")
	}

	for _, reactor := range sw.reactors {
		if err := reactor.AddPeer(peer); err != nil {
			return err
		}
	}
	return nil
}

func (sw *Switch) stopAndRemovePeer(peer *Peer, reason interface{}) {
	sw.peers.Remove(peer)
	for _, reactor := range sw.reactors {
		reactor.RemovePeer(peer, reason)
	}
	peer.Stop()

	sentStatus, receivedStatus := peer.TrafficStatus()
	log.WithFields(log.Fields{
		"module":                logModule,
		"address":               peer.Addr().String(),
		"reason":                reason,
		"duration":              sentStatus.Duration.String(),
		"total_sent":            sentStatus.Bytes,
		"total_received":        receivedStatus.Bytes,
		"average_sent_rate":     sentStatus.AvgRate,
		"average_received_rate": receivedStatus.AvgRate,
		"peer num":              sw.peers.Size(),
	}).Info("disconnect with peer")
}
