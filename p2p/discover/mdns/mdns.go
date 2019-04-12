package mdns

import (
	"context"

	"github.com/grandcat/zeroconf"
	log "github.com/sirupsen/logrus"
)

// Protocol decoration ZeroConf,which is a pure Golang library
// that employs Multicast DNS-SD.
type Protocol struct {
	entries chan *zeroconf.ServiceEntry
	server  *zeroconf.Server
	quite   chan struct{}
}

// NewProtocol create a specific Protocol.
func NewProtocol() *Protocol {
	return &Protocol{
		entries: make(chan *zeroconf.ServiceEntry),
		quite:   make(chan struct{}),
	}
}

func (m *Protocol) getLanPeerLoop(event chan LANPeerEvent) {
	for {
		select {
		case entry := <-m.entries:
			event <- LANPeerEvent{IP: entry.AddrIPv4, Port: entry.Port}
		case <-m.quite:
			return
		}
	}
}

func (m *Protocol) registerService(instance string, service string, domain string, port int) error {
	var err error
	m.server, err = zeroconf.Register(instance, service, domain, port, nil, nil)
	return err
}

func (m *Protocol) registerResolver(event chan LANPeerEvent, service string, domain string) error {
	go m.getLanPeerLoop(event)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns resolver register error")
		return err
	}

	err = resolver.Browse(context.Background(), service, domain, m.entries)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns resolver browse error")
		return err
	}

	return nil
}

func (m *Protocol) stopResolver() {
	close(m.quite)
}

func (m *Protocol) stopService() {
	if m.server != nil {
		m.server.Shutdown()
	}
}
