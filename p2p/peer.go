package p2p

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	crypto "github.com/tendermint/go-crypto"
	wire "github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/config"
)

// peerConn contains the raw connection and its config.
type peerConn struct {
	outbound bool
	config   *PeerConfig
	conn     net.Conn // source connection
}

// Peer represent a bytom network node
type Peer struct {
	cmn.BaseService

	// raw peerConn and the multiplex connection
	*peerConn
	mconn *MConnection // multiplex connection

	*NodeInfo
	Key  string
	Data *cmn.CMap // User data.
}

// PeerConfig is a Peer configuration.
type PeerConfig struct {
	AuthEnc bool `mapstructure:"auth_enc"` // authenticated encryption

	// times are in seconds
	HandshakeTimeout time.Duration `mapstructure:"handshake_timeout"`
	DialTimeout      time.Duration `mapstructure:"dial_timeout"`

	MConfig *MConnConfig `mapstructure:"connection"`

	Fuzz       bool            `mapstructure:"fuzz"` // fuzz connection (for testing)
	FuzzConfig *FuzzConnConfig `mapstructure:"fuzz_config"`
}

// DefaultPeerConfig returns the default config.
func DefaultPeerConfig(config *cfg.P2PConfig) *PeerConfig {
	return &PeerConfig{
		AuthEnc:          true,
		HandshakeTimeout: time.Duration(config.HandshakeTimeout), // * time.Second,
		DialTimeout:      time.Duration(config.DialTimeout),      // * time.Second,
		MConfig:          DefaultMConnConfig(),
		Fuzz:             false,
		FuzzConfig:       DefaultFuzzConnConfig(),
	}
}

func newPeer(pc *peerConn, nodeInfo *NodeInfo, reactorsByCh map[byte]Reactor, chDescs []*ChannelDescriptor, onPeerError func(*Peer, interface{})) *Peer {
	// Key and NodeInfo are set after Handshake
	p := &Peer{
		peerConn: pc,
		NodeInfo: nodeInfo,

		Data: cmn.NewCMap(),
	}
	p.Key = nodeInfo.PubKey.KeyString()
	p.mconn = createMConnection(pc.conn, p, reactorsByCh, chDescs, onPeerError, pc.config.MConfig)

	p.BaseService = *cmn.NewBaseService(nil, "Peer", p)
	return p
}

func newOutboundPeer(addr *NetAddress, reactorsByCh map[byte]Reactor, chDescs []*ChannelDescriptor, onPeerError func(*Peer, interface{}), ourNodePrivKey crypto.PrivKeyEd25519, config *cfg.P2PConfig) (*peerConn, error) {
	return newOutboundPeerConn(addr, reactorsByCh, chDescs, onPeerError, ourNodePrivKey, DefaultPeerConfig(config))
}

func newOutboundPeerConn(addr *NetAddress, reactorsByCh map[byte]Reactor, chDescs []*ChannelDescriptor, onPeerError func(*Peer, interface{}), ourNodePrivKey crypto.PrivKeyEd25519, config *PeerConfig) (*peerConn, error) {
	conn, err := dial(addr, config)
	if err != nil {
		return nil, errors.Wrap(err, "Error dial peer")
	}

	pc, err := newPeerConn(conn, true, reactorsByCh, chDescs, onPeerError, ourNodePrivKey, config)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return pc, nil
}

func newInboundPeerConn(conn net.Conn, reactorsByCh map[byte]Reactor, chDescs []*ChannelDescriptor, onPeerError func(*Peer, interface{}), ourNodePrivKey crypto.PrivKeyEd25519, config *cfg.P2PConfig) (*peerConn, error) {
	return newPeerConn(conn, false, reactorsByCh, chDescs, onPeerError, ourNodePrivKey, DefaultPeerConfig(config))
}

func newPeerConn(rawConn net.Conn, outbound bool, reactorsByCh map[byte]Reactor, chDescs []*ChannelDescriptor, onPeerError func(*Peer, interface{}), ourNodePrivKey crypto.PrivKeyEd25519, config *PeerConfig) (*peerConn, error) {
	conn := rawConn

	// Fuzz connection
	if config.Fuzz {
		// so we have time to do peer handshakes and get set up
		conn = FuzzConnAfterFromConfig(conn, 10*time.Second, config.FuzzConfig)
	}

	// Encrypt connection
	if config.AuthEnc {
		conn.SetDeadline(time.Now().Add(config.HandshakeTimeout * time.Second))

		var err error
		conn, err = MakeSecretConnection(conn, ourNodePrivKey)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating peer")
		}
	}

	// Only the information we already have
	return &peerConn{
		config:   config,
		outbound: outbound,
		conn:     conn,
	}, nil
}

// CloseConn should be used when the peer was created, but never started.
func (pc *peerConn) CloseConn() {
	pc.conn.Close()
}

// HandshakeTimeout performs a handshake between a given node and the peer.
// NOTE: blocking
func (pc *peerConn) HandshakeTimeout(ourNodeInfo *NodeInfo, timeout time.Duration) (*NodeInfo, error) {
	// Set deadline for handshake so we don't block forever on conn.ReadFull
	pc.conn.SetDeadline(time.Now().Add(timeout))

	var peerNodeInfo = new(NodeInfo)
	var err1 error
	var err2 error
	cmn.Parallel(
		func() {
			var n int
			wire.WriteBinary(ourNodeInfo, pc.conn, &n, &err1)
		},
		func() {
			var n int
			wire.ReadBinary(peerNodeInfo, pc.conn, maxNodeInfoSize, &n, &err2)
			log.WithField("peerNodeInfo", peerNodeInfo).Info("Peer handshake")
		})
	if err1 != nil {
		return peerNodeInfo, errors.Wrap(err1, "Error during handshake/write")
	}
	if err2 != nil {
		return peerNodeInfo, errors.Wrap(err2, "Error during handshake/read")
	}

	// Remove deadline
	pc.conn.SetDeadline(time.Time{})

	return peerNodeInfo, nil
}

// Addr returns peer's remote network address.
func (p *Peer) Addr() net.Addr {
	return p.conn.RemoteAddr()
}

// PubKey returns peer's public key.
func (p *Peer) PubKey() crypto.PubKeyEd25519 {
	if p.config.AuthEnc {
		return p.conn.(*SecretConnection).RemotePubKey()
	}
	if p.NodeInfo == nil {
		panic("Attempt to get peer's PubKey before calling Handshake")
	}
	return p.PubKey()
}

// OnStart implements BaseService.
func (p *Peer) OnStart() error {
	p.BaseService.OnStart()
	_, err := p.mconn.Start()
	return err
}

// OnStop implements BaseService.
func (p *Peer) OnStop() {
	p.BaseService.OnStop()
	p.mconn.Stop()
}

// Connection returns underlying MConnection.
func (p *Peer) Connection() *MConnection {
	return p.mconn
}

// IsOutbound returns true if the connection is outbound, false otherwise.
func (p *Peer) IsOutbound() bool {
	return p.outbound
}

// Send msg to the channel identified by chID byte. Returns false if the send
// queue is full after timeout, specified by MConnection.
func (p *Peer) Send(chID byte, msg interface{}) bool {
	if !p.IsRunning() {
		// see Switch#Broadcast, where we fetch the list of peers and loop over
		// them - while we're looping, one peer may be removed and stopped.
		return false
	}
	return p.mconn.Send(chID, msg)
}

// TrySend msg to the channel identified by chID byte. Immediately returns
// false if the send queue is full.
func (p *Peer) TrySend(chID byte, msg interface{}) bool {
	if !p.IsRunning() {
		return false
	}
	return p.mconn.TrySend(chID, msg)
}

// CanSend returns true if the send queue is not full, false otherwise.
func (p *Peer) CanSend(chID byte) bool {
	if !p.IsRunning() {
		return false
	}
	return p.mconn.CanSend(chID)
}

// String representation.
func (p *Peer) String() string {
	if p.outbound {
		return fmt.Sprintf("Peer{%v %v out}", p.mconn, p.Key[:12])
	}

	return fmt.Sprintf("Peer{%v %v in}", p.mconn, p.Key[:12])
}

// Equals reports whenever 2 peers are actually represent the same node.
func (p *Peer) Equals(other *Peer) bool {
	return p.Key == other.Key
}

// Get the data for a given key.
func (p *Peer) Get(key string) interface{} {
	return p.Data.Get(key)
}

func dial(addr *NetAddress, config *PeerConfig) (net.Conn, error) {
	conn, err := addr.DialTimeout(config.DialTimeout * time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func createMConnection(conn net.Conn, p *Peer, reactorsByCh map[byte]Reactor, chDescs []*ChannelDescriptor, onPeerError func(*Peer, interface{}), config *MConnConfig) *MConnection {
	onReceive := func(chID byte, msgBytes []byte) {
		reactor := reactorsByCh[chID]
		if reactor == nil {
			cmn.PanicSanity(cmn.Fmt("Unknown channel %X", chID))
		}
		reactor.Receive(chID, p, msgBytes)
	}

	onError := func(r interface{}) {
		onPeerError(p, r)
	}

	return NewMConnectionWithConfig(conn, chDescs, onReceive, onError, config)
}
