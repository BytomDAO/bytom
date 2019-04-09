package mdns

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"fmt"
	"github.com/bytom/event"
	"sync/atomic"
)

const (
	logModule            = "p2p/mdns"
	registerServiceCycle = 1 * time.Minute
)

type LanPeersEvent struct {
	IP   []net.IP
	Port int
}

type Protocol interface {
	registerService(port int) error
	registerResolver(event chan LanPeersEvent) error
	stopService()
	stopResolver()
}

type LanDiscover struct {
	protocol        Protocol
	resolving       uint32
	servicePort     int
	entries         chan LanPeersEvent
	eventDispatcher *event.Dispatcher
	quite           chan struct{}
}

func NewLanDiscover(protocol Protocol, port int) (*LanDiscover, error) {
	ld := &LanDiscover{
		protocol:        protocol,
		servicePort:     port,
		entries:         make(chan LanPeersEvent, 1024),
		eventDispatcher: event.NewDispatcher(),
		quite:           make(chan struct{}),
	}

	go ld.registerServiceRoutine()
	go ld.getLanPeerLoop()
	return ld, nil
}

func (ld *LanDiscover) Stop() {
	close(ld.quite)
	ld.protocol.stopService()
	ld.protocol.stopResolver()
	ld.eventDispatcher.Stop()
}

func (ld *LanDiscover) Subscribe() (*event.Subscription, error) {
	sub, err := ld.eventDispatcher.Subscribe(LanPeersEvent{})
	if err != nil {
		return nil, err
	}
	if atomic.CompareAndSwapUint32(&ld.resolving, 0, 1) {
		if err = ld.protocol.registerResolver(ld.entries); err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (ld *LanDiscover) registerServiceRoutine() {
	ticker := time.NewTicker(registerServiceCycle)
	defer ticker.Stop()
	err := ld.protocol.registerService(ld.servicePort)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns service register error")
		return
	}
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

func (ld *LanDiscover) getLanPeerLoop() {
	for {
		select {
		case entry := <-ld.entries:
			fmt.Println("====new entry", entry)
			ld.eventDispatcher.Post(entry)
		case <-ld.quite:
			return
		}
	}
}
