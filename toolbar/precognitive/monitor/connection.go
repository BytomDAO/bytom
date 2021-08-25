package monitor

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/p2p"
	"github.com/bytom/bytom/toolbar/precognitive/database/orm"
)

func (m *monitor) connectionRoutine() {
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqMinutes) * time.Minute)
	for ; true; <-ticker.C {
		if err := m.dialNodes(); err != nil {
			log.WithFields(log.Fields{"err": err}).Error("dialNodes")
		}
	}
}

func (m *monitor) dialNodes() error {
	log.Info("Start to reconnect to nodes...")
	var nodes []*orm.Node
	if err := m.db.Model(&orm.Node{}).Find(&nodes).Error; err != nil {
		return err
	}

	addresses := make([]*p2p.NetAddress, 0)
	for i := 0; i < len(nodes); i++ {
		address := p2p.NewNetAddressIPPort(net.ParseIP(nodes[i].IP), nodes[i].Port)
		addresses = append(addresses, address)
	}

	// connected peers will be skipped in switch.DialPeers()
	m.sw.DialPeers(addresses)
	log.Info("DialPeers done.")
	peerList := m.sw.GetPeers().List()
	m.processDialResults(peerList)
	m.checkStatus(peerList)
	return nil
}

func (m *monitor) checkStatus(peerList []*p2p.Peer) {
	for _, peer := range peerList {
		peer.Start()
		m.peers.AddPeer(peer)
	}
	log.WithFields(log.Fields{"num": len(m.sw.GetPeers().List()), "peers": m.sw.GetPeers().List()}).Info("connected peers")

	for _, peerInfo := range m.peers.GetPeerInfos() {
		if peerInfo.Height > m.bestHeightSeen {
			m.bestHeightSeen = peerInfo.Height
		}
	}
	log.WithFields(log.Fields{"bestHeight": m.bestHeightSeen}).Info("peersInfo")
	m.processPeerInfos(m.peers.GetPeerInfos())

	for _, peer := range peerList {
		p := m.peers.GetPeer(peer.ID())
		if p == nil {
			continue
		}

		m.peers.RemovePeer(p.ID())
	}
	log.Info("Disonnect all peers.")
}
