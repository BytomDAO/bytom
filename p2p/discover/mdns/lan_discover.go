package mdns

import (
	"net"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/event"
)

const (
	logModule            = "p2p/mdns"
	registerServiceCycle = 10 * time.Minute
	registerServiceDelay = 2 * time.Second
)

// LANPeerEvent represent LAN peer ip and port.
type LANPeerEvent struct {
	IP   []net.IP
	Port int
}

// mDNSProtocol mdns protocol interface.
type mDNSProtocol interface {
	registerService(port int) error
	registerResolver(event chan LANPeerEvent) error
	stopService()
	stopResolver()
}

// LANDiscover responsible for finding the related services registered LAN nodes.
type LANDiscover struct {
	protocol        mDNSProtocol
	resolving       uint32
	servicePort     int //service port
	entries         chan LANPeerEvent
	eventDispatcher *event.Dispatcher
	quite           chan struct{}
}

// NewLANDiscover create a new LAN node discover.
func NewLANDiscover(protocol mDNSProtocol, port int) *LANDiscover {
	ld := &LANDiscover{
		protocol:        protocol,
		servicePort:     port,
		entries:         make(chan LANPeerEvent, 1024),
		eventDispatcher: event.NewDispatcher(),
		quite:           make(chan struct{}),
	}
	// register service
	go ld.registerServiceRoutine()
	go ld.getLANPeerLoop()
	return ld
}

// Stop stop LAN discover.
func (ld *LANDiscover) Stop() {
	close(ld.quite)
	ld.protocol.stopResolver()
	ld.protocol.stopService()
	ld.eventDispatcher.Stop()
}

// Subscribe used to subscribe for LANPeerEvent.
func (ld *LANDiscover) Subscribe() (*event.Subscription, error) {
	//subscribe LANPeerEvent.
	sub, err := ld.eventDispatcher.Subscribe(LANPeerEvent{})
	if err != nil {
		return nil, err
	}

	//need to register the parser once.
	if atomic.CompareAndSwapUint32(&ld.resolving, 0, 1) {
		if err = ld.protocol.registerResolver(ld.entries); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

// register service routine, will be re-registered periodically
// for the stability of node discovery.
func (ld *LANDiscover) registerServiceRoutine() {
	time.Sleep(registerServiceDelay)
	if err := ld.protocol.registerService(ld.servicePort); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns service register error")
		return
	}

	ticker := time.NewTicker(registerServiceCycle)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ld.protocol.stopService()
			if err := ld.protocol.registerService(ld.servicePort); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns service register error")
				return
			}
		case <-ld.quite:
			return
		}
	}
}

// obtain the lan peer event from the specific protocol
// and distribute it to the subscriber.
func (ld *LANDiscover) getLANPeerLoop() {
	for {
		select {
		case entry := <-ld.entries:
			if err := ld.eventDispatcher.Post(entry); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("event dispatch error")
			}
		case <-ld.quite:
			return
		}
	}
}
