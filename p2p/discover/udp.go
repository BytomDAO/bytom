package discover

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"

	"github.com/bytom/common"
	"github.com/bytom/p2p/netutil"
	"math"
)

const Version = 4

// Errors
var (
	errPacketTooSmall  = errors.New("too small")
	errBadPrefix       = errors.New("bad prefix")
	errChainIDMismatch = errors.New("chainID does not match")
	errPacketType      = errors.New("unknown packet type")
)

// Timeouts
const (
	respTimeout = 1 * time.Second
	expiration  = 20 * time.Second
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
		Rest [][]byte
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
		Rest [][]byte
	}

	// findnode is a query for nodes close to the given target.
	findnode struct {
		Target     NodeID // doesn't need to be an actual public key
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte
	}

	// findnode is a query for nodes close to the given target.
	findnodeHash struct {
		Target     common.Hash
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte
	}

	// reply to findnode
	neighbors struct {
		Nodes      []rpcNode
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte
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
	msgPrefix         = "bytom disv"
	versionPrefixSize = len(msgPrefix)
	nodeIDSize        = 32
	sigSize           = 520 / 8
	chainIDSize       = 8
	headSize          = versionPrefixSize + nodeIDSize + sigSize + chainIDSize // space of packet frame data
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
	priv        *crypto.PrivKeyEd25519
	chainID     string
	ourEndpoint rpcEndpoint
	//nat         nat.Interface
	net netWork
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(priv *crypto.PrivKeyEd25519, chainID string, conn conn, realaddr *net.UDPAddr, nodeDBPath string, netrestrict *netutil.Netlist) (*Network, error) {
	transport, err := listenUDP(priv, chainID, conn, realaddr)
	if err != nil {
		return nil, err
	}
	net, err := newNetwork(transport, priv.PubKey().Unwrap().(crypto.PubKeyEd25519), nodeDBPath, netrestrict)
	if err != nil {
		return nil, err
	}
	log.Info("UDP listener up v5", "net", net.tab.self)
	transport.net = net
	go transport.readLoop()
	return net, nil
}

func listenUDP(priv *crypto.PrivKeyEd25519, chainID string, conn conn, realaddr *net.UDPAddr) (*udp, error) {
	return &udp{conn: conn, priv: priv, chainID: chainID, ourEndpoint: makeEndpoint(realaddr, uint16(realaddr.Port))}, nil
}

func (t *udp) localAddr() *net.UDPAddr {
	return t.conn.LocalAddr().(*net.UDPAddr)
}

func (t *udp) getChainID() string {
	return t.chainID
}

func (t *udp) Close() {
	if err := t.conn.Close(); err != nil {
		log.Error("conn close err:", err)
	}
}

func (t *udp) send(remote *Node, ptype nodeEvent, data interface{}) (hash []byte) {
	hash, err := t.sendPacket(remote.ID, remote.addr(), byte(ptype), data)
	if err != nil {
		log.Error("send packet err:", err)
	}

	return hash
}

func (t *udp) sendPing(remote *Node, toaddr *net.UDPAddr, topics []Topic) (hash []byte) {
	hash, err := t.sendPacket(remote.ID, toaddr, byte(pingPacket), ping{
		Version:    Version,
		From:       t.ourEndpoint,
		To:         makeEndpoint(toaddr, uint16(toaddr.Port)), // TODO: maybe use known TCP port from DB
		Expiration: uint64(time.Now().Add(expiration).Unix()),
		Topics:     topics,
	})
	if err != nil {
		log.Error("send ping packet err:", err)
	}
	return hash
}

func (t *udp) sendFindnode(remote *Node, target NodeID) {
	if _, err := t.sendPacket(remote.ID, remote.addr(), byte(findnodePacket), findnode{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}); err != nil {
		log.Error("send find node packet err:", err)
	}
}

func (t *udp) sendNeighbours(remote *Node, results []*Node) {
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	p := neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	for i, result := range results {
		p.Nodes = append(p.Nodes, nodeToRPC(result))
		if len(p.Nodes) == maxNeighbors || i == len(results)-1 {
			if _, err := t.sendPacket(remote.ID, remote.addr(), byte(neighborsPacket), p); err != nil {
				log.Error("send neighbours packet err:", err)
			}
			p.Nodes = p.Nodes[:0]
		}
	}
}

func (t *udp) sendFindnodeHash(remote *Node, target common.Hash) {
	if _, err := t.sendPacket(remote.ID, remote.addr(), byte(findnodeHashPacket), findnodeHash{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}); err != nil {
		log.Error("send find node hash packet err:", err)
	}
}

func (t *udp) sendTopicRegister(remote *Node, topics []Topic, idx int, pong []byte) {
	if _, err := t.sendPacket(remote.ID, remote.addr(), byte(topicRegisterPacket), topicRegister{
		Topics: topics,
		Idx:    uint(idx),
		Pong:   pong,
	}); err != nil {
		log.Error("send topic register packet err:", err)
	}
}

func (t *udp) sendTopicNodes(remote *Node, queryHash common.Hash, nodes []*Node) {
	p := topicNodes{Echo: queryHash}
	var sent bool
	for _, result := range nodes {
		if result.IP.Equal(t.net.selfIP()) || netutil.CheckRelayIP(remote.IP, result.IP) == nil {
			p.Nodes = append(p.Nodes, nodeToRPC(result))
		}
		if len(p.Nodes) == maxTopicNodes {
			if _, err := t.sendPacket(remote.ID, remote.addr(), byte(topicNodesPacket), p); err != nil {
				log.Error("send topic nodes packet err:", err)
			}
			p.Nodes = p.Nodes[:0]
			sent = true
		}
	}
	if !sent || len(p.Nodes) > 0 {
		if _, err := t.sendPacket(remote.ID, remote.addr(), byte(topicNodesPacket), p); err != nil {
			log.Error("send topic nodes packet err:", err)
		}
	}
}

func (t *udp) sendPacket(toid NodeID, toaddr *net.UDPAddr, ptype byte, req interface{}) (hash []byte, err error) {
	packet, hash, err := encodePacket(t.priv, t.chainID, msgPrefix, ptype, req)
	if err != nil {
		return hash, err
	}
	log.Debug(fmt.Sprintf(">>> %v to %x@%v", nodeEvent(ptype), toid[:8], toaddr))
	if _, err = t.conn.WriteToUDP(packet, toaddr); err != nil {
		log.Info(fmt.Sprint("UDP send failed:", err))
	}
	return hash, err
}

// zeroed padding space for encodePacket.
var headSpace = make([]byte, headSize)

func encodePacket(priv *crypto.PrivKeyEd25519, chainID, msgPrefix string, ptype byte, req interface{}) (p, h []byte, err error) {
	b := new(bytes.Buffer)
	b.Write(headSpace)
	b.WriteByte(ptype)
	var size int
	wire.WriteJSON(req, b, &size, &err)
	if err != nil {
		log.Error(fmt.Sprint("error encoding packet:", err))
		return nil, nil, err
	}
	packet := b.Bytes()
	nodeID := priv.PubKey().Unwrap().(crypto.PubKeyEd25519)
	sig := priv.Sign(hash(packet[headSize:]).Bytes())
	copy(packet, msgPrefix)
	copy(packet[versionPrefixSize:], nodeID[:])
	copy(packet[versionPrefixSize+nodeIDSize:], sig.Bytes())
	copy(packet[versionPrefixSize+nodeIDSize+sigSize:], chainID[0:int(math.Min(float64(len(chainID)), float64(chainIDSize)))])

	h = hash(packet[versionPrefixSize:]).Bytes()
	return packet, h, nil
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
			log.Debug(fmt.Sprintf("Temporary read error: %v", err))
			continue
		} else if err != nil {
			// Shut down the loop for permament errors.
			log.Debug(fmt.Sprintf("Read error: %v", err))
			return
		}
		if err := t.handlePacket(from, buf[:nbytes]); err != nil {
			log.Error("handle packet err:", err)
		}
	}
}

func (t *udp) handlePacket(from *net.UDPAddr, buf []byte) error {
	pkt := ingressPacket{remoteAddr: from}
	if err := decodePacket(t.chainID, buf, &pkt); err != nil {
		log.Debug(fmt.Sprintf("Bad packet from %v: %v", from, err))
		//fmt.Println("bad packet", err)
		return err
	}
	t.net.reqReadPacket(pkt)
	return nil
}

func decodePacket(id string, buffer []byte, pkt *ingressPacket) error {
	if len(buffer) < headSize+1 {
		return errPacketTooSmall
	}
	buf := make([]byte, len(buffer))
	copy(buf, buffer)
	prefix, fromID, chainID, sigdata := buf[:versionPrefixSize], buf[versionPrefixSize:versionPrefixSize+nodeIDSize], buf[versionPrefixSize+nodeIDSize+sigSize:headSize], buf[headSize:]
	if !bytes.Equal(prefix, []byte(msgPrefix)) {
		return errBadPrefix
	}

	if !bytes.Equal([]byte(id), chainID[:len(id)]) {
		return errChainIDMismatch
	}

	pkt.rawData = buf
	pkt.hash = hash(buf[versionPrefixSize:]).Bytes()
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
		return errPacketType
	}
	var err error
	wire.ReadJSON(pkt.data, sigdata[1:], &err)
	if err != nil {
		log.Error("wire readjson err:", err)
	}

	return err
}
