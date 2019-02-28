package discover

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-wire"

	"github.com/bytom/common"
	cfg "github.com/bytom/config"
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/p2p/netutil"
	"github.com/bytom/version"
)

const (
	Version   = 4
	logModule = "discover"
)

// Errors
var (
	errPacketTooSmall   = errors.New("too small")
	errBadPrefix        = errors.New("bad prefix")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errUnknownNode      = errors.New("unknown node")
	errTimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
	errInvalidSeedIP    = errors.New("seed ip is invalid")
	errInvalidSeedPort  = errors.New("seed port is invalid")
)

// Timeouts
const (
	respTimeout = 1 * time.Second
	queryDelay  = 1000 * time.Millisecond
	expiration  = 20 * time.Second

	ntpFailureThreshold = 32               // Continuous timeouts after which to check NTP
	ntpWarningCooldown  = 10 * time.Minute // Minimum amount of time to pass before repeating NTP warning
	driftThreshold      = 10 * time.Second // Allowed clock drift before warning user
)

// ReadPacket is sent to the unhandled channel when it could not be processed
type ReadPacket struct {
	Data []byte
	Addr *net.UDPAddr
}

// Config holds Table-related settings.
type Config struct {
	// These settings are required and configure the UDP listener:
	PrivateKey *ecdsa.PrivateKey

	// These settings are optional:
	AnnounceAddr *net.UDPAddr // local address announced in the DHT
	NodeDBPath   string       // if set, the node database is stored at this filesystem location
	//NetRestrict  *netutil.Netlist  // network whitelist
	Bootnodes []*Node           // list of bootstrap nodes
	Unhandled chan<- ReadPacket // unhandled packets are sent on this channel
}

// RPC request structures
type (
	ping struct {
		Version    uint
		From, To   rpcEndpoint
		Expiration uint64

		// v5
		Topics []Topic

		// Ignore additional fields (for forward compatibility).
		Rest []byte
	}

	// pong is the reply to ping.
	pong struct {
		// This field should mirror the UDP envelope address
		// of the ping packet, which provides a way to discover the
		// the external address (after NAT).
		To rpcEndpoint

		ReplyTok   []byte // This contains the hash of the ping packet.
		Expiration uint64 // Absolute timestamp at which the packet becomes invalid.

		// v5
		TopicHash    common.Hash
		TicketSerial uint32
		WaitPeriods  []uint32

		// Ignore additional fields (for forward compatibility).
		Rest []byte
	}

	// findnode is a query for nodes close to the given target.
	findnode struct {
		Target     NodeID // doesn't need to be an actual public key
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []byte
	}

	// findnode is a query for nodes close to the given target.
	findnodeHash struct {
		Target     common.Hash
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []byte
	}

	// reply to findnode
	neighbors struct {
		Nodes      []rpcNode
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []byte
	}

	topicRegister struct {
		Topics []Topic
		Idx    uint
		Pong   []byte
	}

	topicQuery struct {
		Topic      Topic
		Expiration uint64
	}

	// reply to topicQuery
	topicNodes struct {
		Echo  common.Hash
		Nodes []rpcNode
	}

	rpcNode struct {
		IP  net.IP // len 4 for IPv4 or 16 for IPv6
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
		ID  NodeID
	}

	rpcEndpoint struct {
		IP  net.IP // len 4 for IPv4 or 16 for IPv6
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
	}
)

var (
	versionPrefix     = []byte("bytom discovery")
	versionPrefixSize = len(versionPrefix)
	nodeIDSize        = 32
	sigSize           = 520 / 8
	headSize          = versionPrefixSize + nodeIDSize + sigSize // space of packet frame data
)

// Neighbors replies are sent across multiple packets to
// stay below the 1280 byte limit. We compute the maximum number
// of entries by stuffing a packet until it grows too large.
var maxNeighbors = func() int {
	p := neighbors{Expiration: ^uint64(0)}
	maxSizeNode := rpcNode{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		p.Nodes = append(p.Nodes, maxSizeNode)
		var size int
		var err error
		b := new(bytes.Buffer)
		wire.WriteJSON(p, b, &size, &err)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("cannot encode: " + err.Error())
		}
		if headSize+size+1 >= 1280 {
			return n
		}
	}
}()

var maxTopicNodes = func() int {
	p := topicNodes{}
	maxSizeNode := rpcNode{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		p.Nodes = append(p.Nodes, maxSizeNode)
		var size int
		var err error
		b := new(bytes.Buffer)
		wire.WriteJSON(p, b, &size, &err)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("cannot encode: " + err.Error())
		}
		if headSize+size+1 >= 1280 {
			return n
		}
	}
}()

func makeEndpoint(addr *net.UDPAddr, tcpPort uint16) rpcEndpoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return rpcEndpoint{IP: ip, UDP: uint16(addr.Port), TCP: tcpPort}
}

func (e1 rpcEndpoint) equal(e2 rpcEndpoint) bool {
	return e1.UDP == e2.UDP && e1.TCP == e2.TCP && e1.IP.Equal(e2.IP)
}

func nodeFromRPC(sender *net.UDPAddr, rn rpcNode) (*Node, error) {
	if err := netutil.CheckRelayIP(sender.IP, rn.IP); err != nil {
		return nil, err
	}
	n := NewNode(rn.ID, rn.IP, rn.UDP, rn.TCP)
	err := n.validateComplete()
	return n, err
}

func nodeToRPC(n *Node) rpcNode {
	return rpcNode{ID: n.ID, IP: n.IP, UDP: n.UDP, TCP: n.TCP}
}

type ingressPacket struct {
	remoteID   NodeID
	remoteAddr *net.UDPAddr
	ev         nodeEvent
	hash       []byte
	data       interface{} // one of the RPC structs
	rawData    []byte
}

type conn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

type netWork interface {
	reqReadPacket(pkt ingressPacket)
	selfIP() net.IP
}

// udp implements the RPC protocol.
type udp struct {
	conn        conn
	priv        ed25519.PrivateKey
	ourEndpoint rpcEndpoint
	//nat         nat.Interface
	net netWork
}

func NewDiscover(config *cfg.Config, priv ed25519.PrivateKey, port uint16) (*Network, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("0.0.0.0", strconv.FormatUint(uint64(port), 10)))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	ntab, err := ListenUDP(priv, conn, realaddr, path.Join(config.DBDir(), "discover.db"), nil)
	if err != nil {
		return nil, err
	}
	seeds, err := QueryDNSSeeds(net.LookupHost)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on query dns seeds")
	}

	if config.P2P.Seeds != "" {
		codedSeeds := strings.Split(config.P2P.Seeds, ",")
		for _, codedSeed := range codedSeeds {
			ip, port, err := net.SplitHostPort(codedSeed)
			if err != nil {
				return nil, err
			}

			if validIP := net.ParseIP(ip); validIP == nil {
				return nil, errInvalidSeedIP
			}

			if _, err := strconv.ParseUint(port, 10, 16); err != nil {
				return nil, errInvalidSeedPort
			}

			seeds = append(seeds, codedSeed)
		}
	}

	if len(seeds) == 0 {
		return ntab, nil
	}

	var nodes []*Node
	for _, seed := range seeds {
		version.Status.AddSeed(seed)
		url := "enode://" + hex.EncodeToString(crypto.Sha256([]byte(seed))) + "@" + seed
		nodes = append(nodes, MustParseNode(url))
	}

	if err = ntab.SetFallbackNodes(nodes); err != nil {
		return nil, err
	}
	return ntab, nil
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(priv ed25519.PrivateKey, conn conn, realaddr *net.UDPAddr, nodeDBPath string, netrestrict *netutil.Netlist) (*Network, error) {
	transport, err := listenUDP(priv, conn, realaddr)
	if err != nil {
		return nil, err
	}

	net, err := newNetwork(transport, priv.Public(), nodeDBPath, netrestrict)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"module": logModule, "net": net.tab.self}).Info("UDP listener up v5")
	transport.net = net
	go transport.readLoop()
	return net, nil
}

func listenUDP(priv ed25519.PrivateKey, conn conn, realaddr *net.UDPAddr) (*udp, error) {
	return &udp{conn: conn, priv: priv, ourEndpoint: makeEndpoint(realaddr, uint16(realaddr.Port))}, nil
}

func (t *udp) localAddr() *net.UDPAddr {
	return t.conn.LocalAddr().(*net.UDPAddr)
}

func (t *udp) Close() {
	t.conn.Close()
}

func (t *udp) send(remote *Node, ptype nodeEvent, data interface{}) (hash []byte) {
	hash, _ = t.sendPacket(remote.ID, remote.addr(), byte(ptype), data)
	return hash
}

func (t *udp) sendPing(remote *Node, toaddr *net.UDPAddr, topics []Topic) (hash []byte) {
	hash, _ = t.sendPacket(remote.ID, toaddr, byte(pingPacket), ping{
		Version:    Version,
		From:       t.ourEndpoint,
		To:         makeEndpoint(toaddr, uint16(toaddr.Port)), // TODO: maybe use known TCP port from DB
		Expiration: uint64(time.Now().Add(expiration).Unix()),
		Topics:     topics,
	})
	return hash
}

func (t *udp) sendFindnode(remote *Node, target NodeID) {
	t.sendPacket(remote.ID, remote.addr(), byte(findnodePacket), findnode{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
}

func (t *udp) sendNeighbours(remote *Node, results []*Node) {
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	p := neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	for i, result := range results {
		p.Nodes = append(p.Nodes, nodeToRPC(result))
		if len(p.Nodes) == maxNeighbors || i == len(results)-1 {
			t.sendPacket(remote.ID, remote.addr(), byte(neighborsPacket), p)
			p.Nodes = p.Nodes[:0]
		}
	}
}

func (t *udp) sendFindnodeHash(remote *Node, target common.Hash) {
	t.sendPacket(remote.ID, remote.addr(), byte(findnodeHashPacket), findnodeHash{
		Target:     common.Hash(target),
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
}

func (t *udp) sendTopicRegister(remote *Node, topics []Topic, idx int, pong []byte) {
	t.sendPacket(remote.ID, remote.addr(), byte(topicRegisterPacket), topicRegister{
		Topics: topics,
		Idx:    uint(idx),
		Pong:   pong,
	})
}

func (t *udp) sendTopicNodes(remote *Node, queryHash common.Hash, nodes []*Node) {
	p := topicNodes{Echo: queryHash}
	var sent bool
	for _, result := range nodes {
		if result.IP.Equal(t.net.selfIP()) || netutil.CheckRelayIP(remote.IP, result.IP) == nil {
			p.Nodes = append(p.Nodes, nodeToRPC(result))
		}
		if len(p.Nodes) == maxTopicNodes {
			t.sendPacket(remote.ID, remote.addr(), byte(topicNodesPacket), p)
			p.Nodes = p.Nodes[:0]
			sent = true
		}
	}
	if !sent || len(p.Nodes) > 0 {
		t.sendPacket(remote.ID, remote.addr(), byte(topicNodesPacket), p)
	}
}

func (t *udp) sendPacket(toid NodeID, toaddr *net.UDPAddr, ptype byte, req interface{}) (hash []byte, err error) {
	packet, hash, err := encodePacket(t.priv, ptype, req)
	if err != nil {
		return hash, err
	}
	log.WithFields(log.Fields{"module": logModule, "event": nodeEvent(ptype), "to id": hex.EncodeToString(toid[:8]), "to addr": toaddr}).Debug("send packet")
	if _, err = t.conn.WriteToUDP(packet, toaddr); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Info(fmt.Sprint("UDP send failed"))
	}
	return hash, err
}

// zeroed padding space for encodePacket.
var headSpace = make([]byte, headSize)

func encodePacket(priv ed25519.PrivateKey, ptype byte, req interface{}) (p, hash []byte, err error) {
	b := new(bytes.Buffer)
	b.Write(headSpace)
	b.WriteByte(ptype)
	var size int
	wire.WriteJSON(req, b, &size, &err)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("error encoding packet")
		return nil, nil, err
	}
	packet := b.Bytes()
	nodeID := priv.Public()
	sig := ed25519.Sign(priv, common.BytesToHash(packet[headSize:]).Bytes())
	copy(packet, versionPrefix)
	copy(packet[versionPrefixSize:], nodeID[:])
	copy(packet[versionPrefixSize+nodeIDSize:], sig)

	hash = common.BytesToHash(packet[versionPrefixSize:]).Bytes()
	return packet, hash, nil
}

// readLoop runs in its own goroutine. it injects ingress UDP packets
// into the network loop.
func (t *udp) readLoop() {
	defer t.conn.Close()
	// Discovery packets are defined to be no larger than 1280 bytes.
	// Packets larger than this size will be cut at the end and treated
	// as invalid because their hash won't match.
	buf := make([]byte, 1280)
	for {
		nbytes, from, err := t.conn.ReadFromUDP(buf)
		if netutil.IsTemporaryError(err) {
			// Ignore temporary read errors.
			log.WithFields(log.Fields{"module": logModule, "error": err}).Debug("Temporary read error")
			continue
		} else if err != nil {
			// Shut down the loop for permament errors.
			log.WithFields(log.Fields{"module": logModule, "error": err}).Debug("Read error")
			return
		}
		t.handlePacket(from, buf[:nbytes])
	}
}

func (t *udp) handlePacket(from *net.UDPAddr, buf []byte) error {
	pkt := ingressPacket{remoteAddr: from}
	if err := decodePacket(buf, &pkt); err != nil {
		log.WithFields(log.Fields{"module": logModule, "from": from, "error": err}).Error("Bad packet")
		return err
	}
	t.net.reqReadPacket(pkt)
	return nil
}

func decodePacket(buffer []byte, pkt *ingressPacket) error {
	if len(buffer) < headSize+1 {
		return errPacketTooSmall
	}
	buf := make([]byte, len(buffer))
	copy(buf, buffer)
	prefix, fromID, sigdata := buf[:versionPrefixSize], buf[versionPrefixSize:versionPrefixSize+nodeIDSize], buf[headSize:]
	if !bytes.Equal(prefix, versionPrefix) {
		return errBadPrefix
	}
	pkt.rawData = buf
	pkt.hash = common.BytesToHash(buf[versionPrefixSize:]).Bytes()
	pkt.remoteID = ByteID(fromID)
	switch pkt.ev = nodeEvent(sigdata[0]); pkt.ev {
	case pingPacket:
		pkt.data = new(ping)
	case pongPacket:
		pkt.data = new(pong)
	case findnodePacket:
		pkt.data = new(findnode)
	case neighborsPacket:
		pkt.data = new(neighbors)
	case findnodeHashPacket:
		pkt.data = new(findnodeHash)
	case topicRegisterPacket:
		pkt.data = new(topicRegister)
	case topicQueryPacket:
		pkt.data = new(topicQuery)
	case topicNodesPacket:
		pkt.data = new(topicNodes)
	default:
		return fmt.Errorf("unknown packet type: %d", sigdata[0])
	}
	var err error
	wire.ReadJSON(pkt.data, sigdata[1:], &err)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("wire readjson err")
	}

	return err
}
