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
	"github.com/bytom/p2p/trust"
	"github.com/bytom/errors"
)

const (
	bannedPeerKey      = "BannedPeer"
	defaultBanDuration = time.Hour * 1
)

var (
	ErrDuplicatePeer     = errors.New("Duplicate peer")
	ErrConnectSelf       = errors.New("Connect self")
	ErrConnectBannedPeer = errors.New("Connect banned peer")
)

//-----------------------------------------------------------------------------

// Switch handles peer connections and exposes an API to receive incoming messages
// on `Reactors`.  Each `Reactor` is responsible for handling incoming messages of one
// or more `Channels`.  So while sending outgoing messages is typically performed on the peer,
// incoming messages are received on the reactor.
type Switch struct {
	cmn.BaseService

	config       *cfg.P2PConfig
	peerConfig   *PeerConfig
	listeners    []Listener
	reactors     map[string]Reactor
	chDescs      []*ChannelDescriptor
	reactorsByCh map[byte]Reactor
	peers        *PeerSet
	dialing      *cmn.CMap
	nodeInfo     *NodeInfo             // our node info
	nodePrivKey  crypto.PrivKeyEd25519 // our node privkey
	addrBook     *AddrBook
	bannedPeer   map[string]time.Time
	db           dbm.DB
	mtx          sync.Mutex
}

// NewSwitch creates a new Switch with the given config.
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
	}
	sw.BaseService = *cmn.NewBaseService(nil, "P2P Switch", sw)

	// Optionally, start the pex reactor
	if config.PexReactor {
		sw.addrBook = NewAddrBook(config.AddrBookFile(), config.AddrBookStrict)
		pexReactor := NewPEXReactor(sw.addrBook, sw)
		sw.AddReactor("PEX", pexReactor)
	}

	sw.bannedPeer = make(map[string]time.Time)
	if datajson := sw.db.Get([]byte(bannedPeerKey)); datajson != nil {
		if err := json.Unmarshal(datajson, &sw.bannedPeer); err != nil {
			return nil
		}
	}
	trust.Init()
	return sw
}

// AddReactor adds the given reactor to the switch.
// NOTE: Not goroutine safe.
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

// Reactors returns a map of reactors registered on the switch.
// NOTE: Not goroutine safe.
func (sw *Switch) Reactors() map[string]Reactor {
	return sw.reactors
}

// Reactor returns the reactor with the given name.
// NOTE: Not goroutine safe.
func (sw *Switch) Reactor(name string) Reactor {
	return sw.reactors[name]
}

// AddListener adds the given listener to the switch for listening to incoming peer connections.
// NOTE: Not goroutine safe.
func (sw *Switch) AddListener(l Listener) {
	sw.listeners = append(sw.listeners, l)
}

// Listeners returns the list of listeners the switch listens on.
// NOTE: Not goroutine safe.
func (sw *Switch) Listeners() []Listener {
	return sw.listeners
}

// IsListening returns true if the switch has at least one listener.
// NOTE: Not goroutine safe.
func (sw *Switch) IsListening() bool {
	return len(sw.listeners) > 0
}

// SetNodeInfo sets the switch's NodeInfo for checking compatibility and handshaking with other nodes.
// NOTE: Not goroutine safe.
func (sw *Switch) SetNodeInfo(nodeInfo *NodeInfo) {
	sw.nodeInfo = nodeInfo
}

// NodeInfo returns the switch's NodeInfo.
// NOTE: Not goroutine safe.
func (sw *Switch) NodeInfo() *NodeInfo {
	return sw.nodeInfo
}

// SetNodeKey sets the switch's private key for authenticated encryption.
// NOTE: Not goroutine safe.
func (sw *Switch) SetNodePrivKey(nodePrivKey crypto.PrivKeyEd25519) {
	sw.nodePrivKey = nodePrivKey
	if sw.nodeInfo != nil {
		sw.nodeInfo.PubKey = nodePrivKey.PubKey().Unwrap().(crypto.PubKeyEd25519)
	}
}

// OnStart implements BaseService. It starts all the reactors, peers, and listeners.
func (sw *Switch) OnStart() error {
	// Start reactors
	for _, reactor := range sw.reactors {
		_, err := reactor.Start()
		if err != nil {
			return err
		}
	}
	// Start listeners
	for _, listener := range sw.listeners {
		go sw.listenerRoutine(listener)
	}
	return nil
}

// OnStop implements BaseService. It stops all listeners, peers, and reactors.
func (sw *Switch) OnStop() {
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

// addPeer performs the P2P handshake with a peer
// that already has a SecretConnection. If all goes well,
// it starts the peer and adds it to the switch.
// NOTE: This performs a blocking handshake before the peer is added.
// CONTRACT: If error is returned, peer is nil, and conn is immediately closed.
func (sw *Switch) AddPeer(peer *Peer) error {
	if err := peer.HandshakeTimeout(sw.nodeInfo, time.Duration(sw.peerConfig.HandshakeTimeout*time.Second)); err != nil {
		return ErrConnectBannedPeer
	}
	//filter peer
	if err := sw.filterConnByPeer(peer); err != nil {
		return err
	}
	// Check version, chain id
	if err := sw.nodeInfo.CompatibleWith(peer.NodeInfo); err != nil {
		return err
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

	log.Info("Added peer:", peer)
	return nil
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
func (sw *Switch) DialSeeds(seeds []string) error {
	netAddrs, err := NewNetAddressStrings(seeds)
	if err != nil {
		return err
	}

	if sw.addrBook != nil {
		// add seeds to `addrBook`
		ourAddr, _ := NewNetAddressString(sw.nodeInfo.ListenAddr)
		for _, netAddr := range netAddrs {
			// do not add ourselves
			if netAddr.Equals(ourAddr) {
				continue
			}
			sw.addrBook.AddAddress(netAddr, ourAddr)
		}

		sw.addrBook.Save()
	}

	//permute the list, dial them in random order.
	perm := rand.Perm(len(netAddrs))
	for i := 0; i < len(perm); i += 2 {
		j := perm[i]
		sw.dialSeed(netAddrs[j])
	}

	return nil
}

func (sw *Switch) dialSeed(addr *NetAddress) {
	peer, err := sw.DialPeerWithAddress(addr)
	if err != nil {
		log.WithField("error", err).Error("Error dialing seed")
	} else {
		log.WithField("peer", peer).Info("Connected to seed")
	}
}

func (sw *Switch) addrBookDelSelf() error {
	addr, err := NewNetAddressString(sw.nodeInfo.ListenAddr)
	if err != nil {
		return err
	}
	// remove the given address from the address book if we're added it earlier
	sw.addrBook.RemoveAddress(addr)
	// add the given address to the address book to avoid dialing ourselves
	// again this is our public address
	sw.addrBook.AddOurAddress(addr)
	return nil
}

func (sw *Switch) filterConnByIP(ip string) error {
	if err := sw.checkBannedPeer(ip); err != nil {
		return ErrConnectBannedPeer
	}

	if ip == sw.nodeInfo.ListenHost() {
		sw.addrBookDelSelf()
		return ErrConnectSelf
	}

	return nil
}

func (sw *Switch) filterConnByPeer(peer *Peer) error {
	if err := sw.checkBannedPeer(peer.mconn.RemoteAddress.IP.String()); err != nil {
		return ErrConnectBannedPeer
	}

	if sw.nodeInfo.PubKey.Equals(peer.PubKey().Wrap()) {
		sw.addrBookDelSelf()
		return ErrConnectSelf
	}

	// Check for duplicate peer
	if sw.peers.Has(peer.Key) {
		return ErrDuplicatePeer
	}
	return nil
}

func (sw *Switch) DialPeerWithAddress(addr *NetAddress) (*Peer, error) {
	log.Debug("Dialing peer address:", addr)

	if err := sw.filterConnByIP(addr.IP.String()); err != nil {
		return nil, err
	}

	sw.dialing.Set(addr.IP.String(), addr)
	defer sw.dialing.Delete(addr.IP.String())

	peer, err := newOutboundPeerWithConfig(addr, sw.reactorsByCh, sw.chDescs, sw.StopPeerForError, sw.nodePrivKey, sw.peerConfig)
	if err != nil {
		log.Debug("Failed to dial peer", " address:", addr, " error:", err)
		return nil, err
	}

	err = sw.AddPeer(peer)
	if err != nil {
		log.Info("Failed to add peer:", addr, " err:", err)
		peer.CloseConn()
		return nil, err
	}
	log.Info("Dialed and added peer:", addr)
	return peer, nil
}

func (sw *Switch) IsDialing(addr *NetAddress) bool {
	return sw.dialing.Has(addr.IP.String())
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

// StopPeerForError disconnects from a peer due to external error.
func (sw *Switch) StopPeerForError(peer *Peer, reason interface{}) {
	log.Info("Stopping peer for error.", " peer:", peer, " err:", reason)
	sw.stopAndRemovePeer(peer, reason)
}

// Disconnect from a peer gracefully.
func (sw *Switch) StopPeerGracefully(peer *Peer) {
	log.Info("Stopping peer gracefully")
	sw.stopAndRemovePeer(peer, nil)
}

func (sw *Switch) stopAndRemovePeer(peer *Peer, reason interface{}) {
	for _, reactor := range sw.reactors {
		reactor.RemovePeer(peer, reason)
	}
	sw.peers.Remove(peer)
	peer.Stop()
}

func (sw *Switch) listenerRoutine(l Listener) {
	for {
		inConn, ok := <-l.Connections()
		if !ok {
			break
		}

		// disconnect if we alrady have 2 * MaxNumPeers, we do this because we wanna address book get exchanged even if
		// the connect is full. The pex will disconnect the peer after address exchange, the max connected peer won't
		// be double of MaxNumPeers
		if sw.peers.Size() >= sw.config.MaxNumPeers*2 {
			inConn.Close()
			log.Info("Ignoring inbound connection: already have enough peers.")
			continue
		}

		// New inbound connection!
		err := sw.addPeerWithConnection(inConn)
		if err != nil {
			log.Info("Ignoring inbound connection: error while adding peer.", " address:", inConn.RemoteAddr().String(), " error:", err)
			continue
		}
	}
}

func (sw *Switch) addPeerWithConnection(conn net.Conn) error {
	peer, err := newInboundPeerConn(conn, sw.reactorsByCh, sw.chDescs, sw.StopPeerForError, sw.nodePrivKey, sw.config)
	if err != nil {
		conn.Close()
		return err
	}
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

func (sw *Switch) delBannedPeer(addr string) error {
	delete(sw.bannedPeer, addr)
	datajson, err := json.Marshal(sw.bannedPeer)
	if err != nil {
		return err
	}
	sw.db.Set([]byte(bannedPeerKey), datajson)
	return nil
}

func (sw *Switch) checkBannedPeer(peer string) error {
	sw.mtx.Lock()
	defer sw.mtx.Unlock()

	if banEnd, ok := sw.bannedPeer[peer]; ok {
		if time.Now().Before(banEnd) {
			return ErrConnectBannedPeer
		}
		sw.delBannedPeer(peer)
	}
	return nil
}

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

