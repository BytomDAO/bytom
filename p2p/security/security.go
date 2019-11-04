package security

import (
	log "github.com/sirupsen/logrus"

	cfg "github.com/bytom/bytom/config"
)

const logModule = "p2pSecurity"

type Security struct {
	filter        *PeerFilter
	blacklist     *Blacklist
	peersBanScore *PeersBanScore
}

func NewSecurity(config *cfg.Config) *Security {
	return &Security{
		filter:        NewPeerFilter(),
		blacklist:     NewBlacklist(config),
		peersBanScore: NewPeersScore(),
	}
}

func (s *Security) DoFilter(ip string, pubKey string) error {
	return s.filter.doFilter(ip, pubKey)
}

func (s *Security) IsBanned(ip string, level byte, reason string) bool {
	if ok := s.peersBanScore.Increase(ip, level, reason); !ok {
		return false
	}

	if err := s.blacklist.AddPeer(ip); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on add ban peer")
	}
	//clear peer score
	s.peersBanScore.DelPeer(ip)
	return true
}

func (s *Security) RegisterFilter(filter Filter) {
	s.filter.register(filter)
}

func (s *Security) Start() error {
	if err := s.blacklist.LoadPeers(); err != nil {
		return err
	}

	s.filter.register(s.blacklist)
	return nil
}
