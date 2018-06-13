package pex

import (
	"errors"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/p2p"
	"github.com/bytom/p2p/connection"
	"github.com/bytom/p2p/discover"
)

const (
	// PexChannel is a channel for PEX messages
	PexChannel = byte(0x00)

	minNumOutboundPeers      = 5
	maxPexMessageSize        = 1048576 // 1MB
	defaultMaxMsgCountByPeer = uint16(1000)
)

// PEXReactor handles peer exchange and ensures that an adequate number of peers are connected to the switch.
type PEXReactor struct {
	p2p.BaseReactor
	discv          *discover.Network
	msgCountByPeer *cmn.CMap
}

// NewPEXReactor creates new PEX reactor.
func NewPEXReactor(discv *discover.Network) *PEXReactor {
	r := &PEXReactor{
		discv:          discv,
		msgCountByPeer: cmn.NewCMap(),
	}
	r.BaseReactor = *p2p.NewBaseReactor("PEXReactor", r)
	return r
}

// OnStart implements BaseService
func (r *PEXReactor) OnStart() error {
	r.BaseReactor.OnStart()
	go r.ensurePeersRoutine()
	go r.flushMsgCountByPeer()
	return nil
}

// OnStop implements BaseService
func (r *PEXReactor) OnStop() {
	r.BaseReactor.OnStop()
}

// GetChannels implements Reactor
func (r *PEXReactor) GetChannels() []*connection.ChannelDescriptor {
	return []*connection.ChannelDescriptor{&connection.ChannelDescriptor{
		ID:                PexChannel,
		Priority:          1,
		SendQueueCapacity: 10,
	}}
}

// AddPeer adding peer to the address book
func (r *PEXReactor) AddPeer(p *p2p.Peer) error {
	if r.Switch.Peers().Size() <= r.Switch.Config.P2P.MaxNumPeers {
		return nil
	}

	nodes := make([]*discover.Node, 20)
	if n := r.discv.ReadRandomNodes(nodes); n == 0 {
		return nil
	}

	if r.SendAddrs(p, nodes) {
		<-time.After(1 * time.Second)
		r.Switch.StopPeerGracefully(p)
	}
	return errors.New("addPeer: reach the max peer, exchange then close")
}

// Receive implements Reactor by handling incoming PEX messages.
func (r *PEXReactor) Receive(chID byte, p *p2p.Peer, rawMsg []byte) {
	r.incrementMsgCount(p.RemoteAddr)
	if r.reachedMaxMsgLimit(p.RemoteAddr) {
		log.WithField("peer", p.RemoteAddr).Error("reached the max pex messages limit")
		r.Switch.StopPeerGracefully(p)
		return
	}

	_, msg, err := DecodeMessage(rawMsg)
	if err != nil {
		log.WithField("error", err).Error("failed to decoding pex message")
		r.Switch.StopPeerGracefully(p)
		return
	}

	switch msg := msg.(type) {
	case *pexRequestMessage:
		nodes := make([]*discover.Node, 20)
		if n := r.discv.ReadRandomNodes(nodes); n == 0 {
			return
		}

		if !r.SendAddrs(p, nodes) {
			log.Error("failed to send pex address message")
		}

	case *pexAddrsMessage:
	default:
		log.WithField("type", reflect.TypeOf(msg)).Error("Unknown message type")
	}
}

// RemovePeer implements Reactor.
func (r *PEXReactor) RemovePeer(p *p2p.Peer, reason interface{}) {}

// SendAddrs sends addrs to the peer.
func (r *PEXReactor) SendAddrs(p *p2p.Peer, nodes []*discover.Node) bool {
	addrs := []*p2p.NetAddress{}
	for _, node := range nodes {
		if node == nil {
			break
		}
		addrs = append(addrs, p2p.NewNetAddressIPPort(node.IP, node.TCP))
	}

	ok := p.TrySend(PexChannel, struct{ PexMessage }{&pexAddrsMessage{Addrs: addrs}})
	if !ok {
		r.Switch.StopPeerGracefully(p)
	}
	return ok
}

func (r *PEXReactor) dialPeerWorker(a *p2p.NetAddress, wg *sync.WaitGroup) {
	if err := r.Switch.DialPeerWithAddress(a); err != nil {
		log.WithFields(log.Fields{"addr": a, "err": err}).Error("dialPeerWorker fail on dial peer")
	}
	wg.Done()
}

func (r *PEXReactor) ensurePeers() {
	numOutPeers, _, numDialing := r.Switch.NumPeers()
	numToDial := (minNumOutboundPeers - (numOutPeers + numDialing))
	log.WithFields(log.Fields{
		"numOutPeers": numOutPeers,
		"numDialing":  numDialing,
		"numToDial":   numToDial,
	}).Debug("ensure peers")
	if numToDial <= 0 {
		return
	}

	toDial := make(map[string]*p2p.NetAddress)
	connectedPeers := make(map[string]struct{})
	for _, peer := range r.Switch.Peers().List() {
		connectedPeers[peer.RemoteAddrHost()] = struct{}{}
	}

	nodes := make([]*discover.Node, numToDial)
	n := r.discv.ReadRandomNodes(nodes)
	for i := 0; i < n && len(toDial) < numToDial; i++ {
		try := p2p.NewNetAddressIPPort(nodes[i].IP, nodes[i].TCP)
		if r.Switch.NodeInfo().RemoteAddr == try.String() {
			continue
		}
		if _, selected := toDial[try.IP.String()]; selected {
			continue
		}
		if dialling := r.Switch.IsDialing(try); dialling {
			continue
		}
		if _, ok := connectedPeers[try.IP.String()]; ok {
			continue
		}

		log.Debug("Will dial address addr:", try)
		toDial[try.IP.String()] = try
	}

	var wg sync.WaitGroup
	for _, item := range toDial {
		wg.Add(1)
		go r.dialPeerWorker(item, &wg)
	}
	wg.Wait()
}

func (r *PEXReactor) ensurePeersRoutine() {
	r.ensurePeers()
	ticker := time.NewTicker(120 * time.Second)
	quickTicker := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-ticker.C:
			r.ensurePeers()
		case <-quickTicker.C:
			if r.Switch.Peers().Size() < 3 {
				r.ensurePeers()
			}
		case <-r.Quit:
			return
		}
	}
}

func (r *PEXReactor) flushMsgCountByPeer() {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ticker.C:
			r.msgCountByPeer.Clear()
		case <-r.Quit:
			return
		}
	}
}

func (r *PEXReactor) incrementMsgCount(addr string) {
	var count uint16
	if countI := r.msgCountByPeer.Get(addr); countI != nil {
		count = countI.(uint16)
	}
	count++
	r.msgCountByPeer.Set(addr, count)
}

func (r *PEXReactor) reachedMaxMsgLimit(addr string) bool {
	return r.msgCountByPeer.Get(addr).(uint16) >= defaultMaxMsgCountByPeer
}
