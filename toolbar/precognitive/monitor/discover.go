package monitor

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/p2p/discover/dht"
	"github.com/bytom/bytom/toolbar/precognitive/config"
)

var (
	nodesToDiscv = 150
	discvFreqSec = 60
)

func (m *monitor) discoveryRoutine() {
	discvMap := make(map[string]*dht.Node)
	ticker := time.NewTicker(time.Duration(discvFreqSec) * time.Second)
	for range ticker.C {
		nodes := make([]*dht.Node, nodesToDiscv)
		num := m.sw.GetDiscv().ReadRandomNodes(nodes)
		for _, node := range nodes[:num] {
			address := fmt.Sprintf("%s:%d", node.IP.String(), node.TCP)
			if n, ok := discvMap[address]; ok && n.String() == node.String() {
				continue
			}

			log.WithFields(log.Fields{"new node": node}).Info("discover")

			if err := m.upsertNode(&config.Node{
				PublicKey: node.ID.String(),
				IP:        node.IP.String(),
				Port:      node.TCP,
			}); err != nil {
				log.WithFields(log.Fields{"node": node, "err": err}).Error("upsertNode")
			} else {
				discvMap[address] = node
			}
		}
	}
}
