package p2p

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	crypto "github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	cfg "github.com/bytom/config"
	"github.com/bytom/errors"
	"github.com/bytom/p2p/trust"
)

const (
	reconnectAttempts = 30
	reconnectInterval = 3 * time.Second

	bannedPeerKey      = "BannedPeer"
	defaultBanDuration = time.Hour * 24
	peerBannedTM       = 20
)

var ErrConnectBannedPeer = errors.New("Connect banned peer")

type Reactor interface {
	cmn.Service // Start, Stop

	SetSwitch(*Switch)
	GetChannels() []*ChannelDescriptor
	AddPeer(peer *Peer) error
	RemovePeer(peer *Peer, reason interface{})
	Receive(chID byte, peer *Peer, msgBytes []byte)
}

//--------------------------------------

type BaseReactor struct {
	cmn.BaseService // Provides Start, Stop, .Quit
	Switch          *Switch
}

func NewBaseReactor(name string, impl Reactor) *BaseReactor {
	return &BaseReactor{
		BaseService: *cmn.NewBaseService(nil, name, impl),
		Switch:      nil,
	}
}

func (br *BaseReactor) SetSwitch(sw *Switch) {
	br.Switch = sw
}
func (_ *BaseReactor) GetChannels() []*ChannelDescriptor              { return nil }
func (_ *BaseReactor) AddPeer(peer *Peer)                             {}
func (_ *BaseReactor) RemovePeer(peer *Peer, reason interface{})      {}
func (_ *BaseReactor) Receive(chID byte, peer *Peer, msgBytes []byte) {}

//-----------------------------------------------------------------------------

/*
The `Switch` handles peer connections and exposes an API to receive incoming messages
on `Reactors`.  Each `Reactor` is responsible for handling incoming messages of one
or more `Channels`.  So while sending outgoing messages is typically performed on the peer,
incoming messages are received on the reactor.
*/
type Switch struct {
	cmn.BaseService

	config           *cfg.P2PConfig
	peerConfig       *PeerConfig
	listeners        []Listener
	reactors         map[string]Reactor
	chDescs          []*ChannelDescriptor
	reactorsByCh     map[byte]Reactor
	peers            *PeerSet
	dialing          *cmn.CMap
	nodeInfo         *NodeInfo             // our node info
	nodePrivKey      crypto.PrivKeyEd25519 // our node privkey
	bannedPeer       map[string]time.Time
	db               dbm.DB
	TrustMetricStore *trust.TrustMetricStore
	ScamPeerCh       chan *Peer
	mtx              sync.Mutex

	filterConnByAddr   func(net.Addr) error
	filterConnByPubKey func(crypto.PubKeyEd25519) error
}

var (
	ErrSwitchDuplicatePeer = errors.New("Duplicate peer")
)

func NewSwitch(config *cfg.P2PConfig, trustHistoryDB dbm.DB) *Switch {
	sw := &Switch{
		config:       config,
		peerConfig:   DefaultPeerConfig(config),
		reactors:     make(map[string]Reactor),
		chDescs:      make([]*ChannelDescriptor, 0),
		reactorsByCh: make(map[byte]Reactor),
		peers:        NewPeerSet(),
		dialing:      cmn.NewCMap(),
		nodeInfo:     nil,
		db:           trustHistoryDB,
		ScamPeerCh:   make(chan *Peer),
	}
	sw.BaseService = *cmn.NewBaseService(nil, "P2P Switch", sw)
	sw.TrustMetricStore = trust.NewTrustMetricStore(trustHistoryDB, trust.DefaultConfig())
	sw.TrustMetricStore.Start()

	sw.bannedPeer = make(map[string]time.Time)
	if datajson := sw.db.Get([]byte(bannedPeerKey)); datajson != nil {
		if err := json.Unmarshal(datajson, &sw.bannedPeer); err != nil {
			return nil
		}
	}
	go sw.scamPeerHandler()
	return sw
}

// Not goroutine safe.
func (sw *Switch) AddReactor(name string, reactor Reactor) Reactor {
	// Validate the reactor.
	// No two reactors can share the same channel.
	reactorChannels := reactor.GetChannels()
	for _, chDesc := range reactorChannels {
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

// Not goroutine safe.
func (sw *Switch) Reactors() map[string]Reactor {
	return sw.reactors
}

// Not goroutine safe.
func (sw *Switch) Reactor(name string) Reactor {
	return sw.reactors[name]
}

// Not goroutine safe.
func (sw *Switch) AddListener(l Listener) {
	sw.listeners = append(sw.listeners, l)
}

// Not goroutine safe.
func (sw *Switch) Listeners() []Listener {
	return sw.listeners
}

// Not goroutine safe.
func (sw *Switch) IsListening() bool {
	return len(sw.listeners) > 0
}

// Not goroutine safe.
func (sw *Switch) SetNodeInfo(nodeInfo *NodeInfo) {
	sw.nodeInfo = nodeInfo
}

// Not goroutine safe.
func (sw *Switch) NodeInfo() *NodeInfo {
	return sw.nodeInfo
}

// Not goroutine safe.
// NOTE: Overwrites sw.nodeInfo.PubKey
func (sw *Switch) SetNodePrivKey(nodePrivKey crypto.PrivKeyEd25519) {
	sw.nodePrivKey = nodePrivKey
	if sw.nodeInfo != nil {
		sw.nodeInfo.PubKey = nodePrivKey.PubKey().Unwrap().(crypto.PubKeyEd25519)
	}
}

// Switch.Start() starts all the reactors, peers, and listeners.
func (sw *Switch) OnStart() error {
	sw.BaseService.OnStart()
	// Start reactors
	for _, reactor := range sw.reactors {
		_, err := reactor.Start()
		if err != nil {
			return err
		}
	}
	// Start peers
	for _, peer := range sw.peers.List() {
		sw.startInitPeer(peer)
	}
	// Start listeners
	for _, listener := range sw.listeners {
		go sw.listenerRoutine(listener)
	}
	return nil
}

func (sw *Switch) OnStop() {
	sw.BaseService.OnStop()
	// Stop listeners
	for _, listener := range sw.listeners {
		listener.Stop()
	}
	sw.listeners = nil
	// Stop peers
	for _, peer := range sw.peers.List() {
		peer.Stop()
		sw.peers.Remove(peer)
	}
	// Stop reactors
	for _, reactor := range sw.reactors {
		reactor.Stop()
	}
}

// NOTE: This performs a blocking handshake before the peer is added.
// CONTRACT: If error is returned, peer is nil, and conn is immediately closed.
func (sw *Switch) AddPeer(peer *Peer) error {
	if err := sw.FilterConnByAddr(peer.Addr()); err != nil {
		return err
	}

	if err := sw.FilterConnByPubKey(peer.PubKey()); err != nil {
		return err
	}

	if err := peer.HandshakeTimeout(sw.nodeInfo, time.Duration(sw.peerConfig.HandshakeTimeout*time.Second)); err != nil {
		return err
	}

	if err := sw.checkBannedPeer(peer.NodeInfo.ListenHost()); err != nil {
		return err
	}

	// Avoid self
	if sw.nodeInfo.PubKey.Equals(peer.PubKey().Wrap()) {
		return errors.New("Ignoring connection from self")
	}

	// Check version, chain id
	if err := sw.nodeInfo.CompatibleWith(peer.NodeInfo); err != nil {
		return err
	}

	// Check for duplicate peer
	if sw.peers.Has(peer.Key) {
		return ErrSwitchDuplicatePeer

	}

	// Start peer
	if sw.IsRunning() {
		if err := sw.startInitPeer(peer); err != nil {
			return err
		}
	}

	// Add the peer to .peers.
	// We start it first so that a peer in the list is safe to Stop.
	// It should not err since we already checked peers.Has()
	if err := sw.peers.Add(peer); err != nil {
		return err
	}

	tm := trust.NewMetric()

	tm.Start()
	sw.TrustMetricStore.AddPeerTrustMetric(peer.mconn.RemoteAddress.IP.String(), tm)

	log.WithField("peer", peer).Info("Added peer")
	return nil
}

func (sw *Switch) FilterConnByAddr(addr net.Addr) error {
	if sw.filterConnByAddr != nil {
		return sw.filterConnByAddr(addr)
	}
	return nil
}

func (sw *Switch) FilterConnByPubKey(pubkey crypto.PubKeyEd25519) error {
	if sw.filterConnByPubKey != nil {
		return sw.filterConnByPubKey(pubkey)
	}
	return nil

}

func (sw *Switch) SetAddrFilter(f func(net.Addr) error) {
	sw.filterConnByAddr = f
}

func (sw *Switch) SetPubKeyFilter(f func(crypto.PubKeyEd25519) error) {
	sw.filterConnByPubKey = f
}

func (sw *Switch) startInitPeer(peer *Peer) error {
	peer.Start() // spawn send/recv routines
	for _, reactor := range sw.reactors {
		if err := reactor.AddPeer(peer); err != nil {
			return err
		}
	}
	return nil
}

// Dial a list of seeds asynchronously in random order
func (sw *Switch) DialSeeds(addrBook *AddrBook, seeds []string) error {

	netAddrs, err := NewNetAddressStrings(seeds)
	if err != nil {
		return err
	}

	if addrBook != nil {
		// add seeds to `addrBook`
		ourAddrS := sw.nodeInfo.ListenAddr
		ourAddr, _ := NewNetAddressString(ourAddrS)
		for _, netAddr := range netAddrs {
			// do not add ourselves
			if netAddr.Equals(ourAddr) {
				continue
			}
			addrBook.AddAddress(netAddr, ourAddr)
		}
		addrBook.Save()
	}

	// permute the list, dial them in random order.
	perm := rand.Perm(len(netAddrs))
	for i := 0; i < len(perm); i++ {
		go func(i int) {
			time.Sleep(time.Duration(rand.Int63n(3000)) * time.Millisecond)
			j := perm[i]
			sw.dialSeed(netAddrs[j])
		}(i)
	}
	return nil
}

func (sw *Switch) dialSeed(addr *NetAddress) {
	peer, err := sw.DialPeerWithAddress(addr, true)
	if err != nil {
		log.WithField("error", err).Error("Error dialing seed")
	} else {
		log.WithField("peer", peer).Info("Connected to seed")
	}
}

func (sw *Switch) DialPeerWithAddress(addr *NetAddress, persistent bool) (*Peer, error) {
	if err := sw.checkBannedPeer(addr.IP.String()); err != nil {
		return nil, err
	}

	sw.dialing.Set(addr.IP.String(), addr)
	defer sw.dialing.Delete(addr.IP.String())

	log.WithField("address", addr).Info("Dialing peer")
	peer, err := newOutboundPeerWithConfig(addr, sw.reactorsByCh, sw.chDescs, sw.StopPeerForError, sw.nodePrivKey, sw.peerConfig)
	if err != nil {
		log.WithFields(log.Fields{
			"address": addr,
			"error":   err,
		}).Info("Failed to dial peer")
		return nil, err
	}
	peer.SetLogger(sw.Logger.With("peer", addr))
	if persistent {
		peer.makePersistent()
	}
	err = sw.AddPeer(peer)
	if err != nil {
		log.WithFields(log.Fields{
			"address": addr,
			"error":   err,
		}).Info("Failed to add peer")
		peer.CloseConn()
		return nil, err
	}
	log.WithFields(log.Fields{
		"address": addr,
	}).Info("Dialed and added peer")
	return peer, nil
}

func (sw *Switch) IsDialing(addr *NetAddress) bool {
	return sw.dialing.Has(addr.IP.String())
}

// Broadcast runs a go routine for each attempted send, which will block
// trying to send for defaultSendTimeoutSeconds. Returns a channel
// which receives success values for each attempted send (false if times out)
// NOTE: Broadcast uses goroutines, so order of broadcast may not be preserved.
func (sw *Switch) Broadcast(chID byte, msg interface{}) chan bool {
	successChan := make(chan bool, len(sw.peers.List()))
	log.WithFields(log.Fields{
		"chID": chID,
		"msg":  msg,
	}).Debug("Broadcast")
	for _, peer := range sw.peers.List() {
		go func(peer *Peer) {
			success := peer.Send(chID, msg)
			successChan <- success
		}(peer)
	}
	return successChan
}

// Returns the count of outbound/inbound and outbound-dialing peers.
func (sw *Switch) NumPeers() (outbound, inbound, dialing int) {
	peers := sw.peers.List()
	for _, peer := range peers {
		if peer.outbound {
			outbound++
		} else {
			inbound++
		}
	}
	dialing = sw.dialing.Size()
	return
}

func (sw *Switch) Peers() *PeerSet {
	return sw.peers
}

// Disconnect from a peer due to external error, retry if it is a persistent peer.
// TODO: make record depending on reason.
func (sw *Switch) StopPeerForError(peer *Peer, reason interface{}) {
	addr := NewNetAddress(peer.Addr())
	log.WithFields(log.Fields{
		"peer":  peer,
		"error": reason,
	}).Info("Stopping peer due to error")
	sw.stopAndRemovePeer(peer, reason)

	if peer.IsPersistent() {
		go func() {
			log.WithField("peer", peer).Info("Reconnecting to peer")
			for i := 1; i < reconnectAttempts; i++ {
				if !sw.IsRunning() {
					return
				}

				peer, err := sw.DialPeerWithAddress(addr, true)
				if err != nil {
					if i == reconnectAttempts {
						log.WithFields(log.Fields{
							"retries": i,
							"error":   err,
						}).Info("Error reconnecting to peer. Giving up")
						return
					}
					if errors.Root(err) == ErrSwitchDuplicatePeer {
						log.WithField("error", err).Info("Error reconnecting to peer. ")
						return
					}
					log.WithFields(log.Fields{
						"retries": i,
						"error":   err,
					}).Info("Error reconnecting to peer. Trying again")
					time.Sleep(reconnectInterval)
					continue
				}

				log.WithField("peer", peer).Info("Reconnected to peer")
				return
			}
		}()
	}
}

// Disconnect from a peer gracefully.
// TODO: handle graceful disconnects.
func (sw *Switch) StopPeerGracefully(peer *Peer) {
	log.Info("Stopping peer gracefully")
	sw.stopAndRemovePeer(peer, nil)
}

func (sw *Switch) stopAndRemovePeer(peer *Peer, reason interface{}) {
	sw.peers.Remove(peer)
	peer.Stop()
	for _, reactor := range sw.reactors {
		reactor.RemovePeer(peer, reason)
	}
}

func (sw *Switch) listenerRoutine(l Listener) {
	for {
		inConn, ok := <-l.Connections()
		if !ok {
			break
		}

		// ignore connection if we already have enough
		maxPeers := sw.config.MaxNumPeers
		if maxPeers <= sw.peers.Size() {
			log.WithFields(log.Fields{
				"address":  inConn.RemoteAddr().String(),
				"numPeers": sw.peers.Size(),
				"max":      maxPeers,
			}).Info("Ignoring inbound connection: already have enough peers")
			continue
		}

		// New inbound connection!
		err := sw.addPeerWithConnectionAndConfig(inConn, sw.peerConfig)
		if err != nil {
			log.WithFields(log.Fields{
				"address": inConn.RemoteAddr().String(),
				"error":   err,
			}).Info("Ignoring inbound connection: error while adding peer")
			continue
		}

		// NOTE: We don't yet have the listening port of the
		// remote (if they have a listener at all).
		// The peerHandshake will handle that
	}

	// cleanup
}

//-----------------------------------------------------------------------------

type SwitchEventNewPeer struct {
	Peer *Peer
}

type SwitchEventDonePeer struct {
	Peer  *Peer
	Error interface{}
}

//------------------------------------------------------------------
// Switches connected via arbitrary net.Conn; useful for testing

// Returns n switches, connected according to the connect func.
// If connect==Connect2Switches, the switches will be fully connected.
// initSwitch defines how the ith switch should be initialized (ie. with what reactors).
// NOTE: panics if any switch fails to start.
func MakeConnectedSwitches(cfg *cfg.P2PConfig, n int, initSwitch func(int, *Switch) *Switch, connect func([]*Switch, int, int)) []*Switch {
	switches := make([]*Switch, n)
	for i := 0; i < n; i++ {
		switches[i] = makeSwitch(cfg, i, "testing", "123.123.123", initSwitch)
	}

	if err := StartSwitches(switches); err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			connect(switches, i, j)
		}
	}

	return switches
}

var PanicOnAddPeerErr = false

// Will connect switches i and j via net.Pipe()
// Blocks until a conection is established.
// NOTE: caller ensures i and j are within bounds
func Connect2Switches(switches []*Switch, i, j int) {
	switchI := switches[i]
	switchJ := switches[j]
	c1, c2 := net.Pipe()
	doneCh := make(chan struct{})
	go func() {
		err := switchI.addPeerWithConnection(c1)
		if PanicOnAddPeerErr && err != nil {
			panic(err)
		}
		doneCh <- struct{}{}
	}()
	go func() {
		err := switchJ.addPeerWithConnection(c2)
		if PanicOnAddPeerErr && err != nil {
			panic(err)
		}
		doneCh <- struct{}{}
	}()
	<-doneCh
	<-doneCh
}

func StartSwitches(switches []*Switch) error {
	for _, s := range switches {
		_, err := s.Start() // start switch and reactors
		if err != nil {
			return err
		}
	}
	return nil
}

func makeSwitch(cfg *cfg.P2PConfig, i int, network, version string, initSwitch func(int, *Switch) *Switch) *Switch {
	privKey := crypto.GenPrivKeyEd25519()
	// new switch, add reactors
	// TODO: let the config be passed in?
	s := initSwitch(i, NewSwitch(cfg, nil))
	s.SetNodeInfo(&NodeInfo{
		PubKey:     privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker:    cmn.Fmt("switch%d", i),
		Network:    network,
		Version:    version,
		RemoteAddr: cmn.Fmt("%v:%v", network, rand.Intn(64512)+1023),
		ListenAddr: cmn.Fmt("%v:%v", network, rand.Intn(64512)+1023),
	})
	s.SetNodePrivKey(privKey)
	return s
}

func (sw *Switch) addPeerWithConnection(conn net.Conn) error {
	peer, err := newInboundPeer(conn, sw.reactorsByCh, sw.chDescs, sw.StopPeerForError, sw.nodePrivKey, sw.config)
	if err != nil {
		conn.Close()
		return err
	}
	peer.SetLogger(sw.Logger.With("peer", conn.RemoteAddr()))
	if err = sw.AddPeer(peer); err != nil {
		conn.Close()
		return err
	}

	return nil
}

func (sw *Switch) addPeerWithConnectionAndConfig(conn net.Conn, config *PeerConfig) error {
	fullAddr := conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(fullAddr)
	if err != nil {
		return err
	}

	if err = sw.checkBannedPeer(host); err != nil {
		return err
	}

	peer, err := newInboundPeerWithConfig(conn, sw.reactorsByCh, sw.chDescs, sw.StopPeerForError, sw.nodePrivKey, config)
	if err != nil {
		conn.Close()
		return err
	}
	peer.SetLogger(sw.Logger.With("peer", conn.RemoteAddr()))
	if err = sw.AddPeer(peer); err != nil {
		conn.Close()
		return err
	}

	return nil
}

func (sw *Switch) AddBannedPeer(peer *Peer) error {
	sw.mtx.Lock()
	defer sw.mtx.Unlock()

	key := peer.mconn.RemoteAddress.IP.String()
	sw.bannedPeer[key] = time.Now().Add(defaultBanDuration)
	datajson, err := json.Marshal(sw.bannedPeer)
	if err != nil {
		return err
	}
	sw.db.Set([]byte(bannedPeerKey), datajson)
	return nil
}

func (sw *Switch) DelBannedPeer(addr string) error {
	sw.mtx.Lock()
	defer sw.mtx.Unlock()

	delete(sw.bannedPeer, addr)
	datajson, err := json.Marshal(sw.bannedPeer)
	if err != nil {
		return err
	}
	sw.db.Set([]byte(bannedPeerKey), datajson)
	return nil
}

func (sw *Switch) scamPeerHandler() {
	for src := range sw.ScamPeerCh {
		var tm *trust.TrustMetric
		key := src.Connection().RemoteAddress.IP.String()
		if tm = sw.TrustMetricStore.GetPeerTrustMetric(key); tm == nil {
			log.Errorf("Can't get peer trust metric")
			continue
		}
		sw.delTrustMetric(tm, src)
	}
}

func (sw *Switch) AddScamPeer(src *Peer) {
	sw.ScamPeerCh <- src
}

func (sw *Switch) delTrustMetric(tm *trust.TrustMetric, src *Peer) {
	key := src.Connection().RemoteAddress.IP.String()
	tm.BadEvents(1)
	if tm.TrustScore() < peerBannedTM {
		sw.AddBannedPeer(src)
		sw.TrustMetricStore.PeerDisconnected(key)
		sw.StopPeerGracefully(src)
	}
}

func (sw *Switch) checkBannedPeer(peer string) error {
	if banEnd, ok := sw.bannedPeer[peer]; ok {
		if time.Now().Before(banEnd) {
			return ErrConnectBannedPeer
		}
		sw.DelBannedPeer(peer)
	}
	return nil
}
