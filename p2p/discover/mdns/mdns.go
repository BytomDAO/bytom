package mdns

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/zeroconf"
)

type mdnsProtocol struct {
	entries chan *zeroconf.ServiceEntry
	server  *zeroconf.Server
	quite   chan struct{}
}

func NewMdnsProtocol() *mdnsProtocol {
	return &mdnsProtocol{
		entries: make(chan *zeroconf.ServiceEntry),
		quite:   make(chan struct{}),
	}
}

func (m *mdnsProtocol) getLanPeerLoop(event chan LanPeersEvent) {
	for {
		select {
		case entry := <-m.entries:
			event <- LanPeersEvent{IP: entry.AddrIPv4, Port: entry.Port}
		case <-m.quite:
			return
		}
	}
}

func (m *mdnsProtocol) registerService(port int) error {
	var err error
	if m.server, err = zeroconf.Register("bytomd", "lanDiscv", "local.", port, nil, nil); err != nil {
		return err
	}
	return nil
}

func (m *mdnsProtocol) registerResolver(event chan LanPeersEvent) error {
	go m.getLanPeerLoop(event)
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns resolver register error")
		return err
	}

	ctx := context.Background()
	err = resolver.Browse(ctx, "lanDiscv", "local.", m.entries)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("mdns resolver browse error")
		return err
	}
	return nil
}

// 如何防止多次关闭chan
func (m *mdnsProtocol) stopResolver() {
	close(m.quite)
}

func (m *mdnsProtocol) stopService() {
	if m.server != nil {
		m.server.Shutdown()
	}
}
