package security

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	defaultBanThreshold  = uint32(100)
	defaultWarnThreshold = uint32(50)

	LevelMsgIllegal              = 0x01
	levelMsgIllegalPersistent    = uint32(20)
	levelMsgIllegalTransient     = uint32(0)
	LevelConnException           = 0x02
	levelConnExceptionPersistent = uint32(0)
	levelConnExceptionTransient  = uint32(20)
)

type PeersBanScore struct {
	peers map[string]*DynamicBanScore
	mtx   sync.Mutex
}

func NewPeersScore() *PeersBanScore {
	return &PeersBanScore{
		peers: make(map[string]*DynamicBanScore),
	}
}

func (ps *PeersBanScore) DelPeer(ip string) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	delete(ps.peers, ip)
}

func (ps *PeersBanScore) Increase(ip string, level byte, reason string) bool {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	var persistent, transient uint32
	switch level {
	case LevelMsgIllegal:
		persistent = levelMsgIllegalPersistent
		transient = levelMsgIllegalTransient
	case LevelConnException:
		persistent = levelConnExceptionPersistent
		transient = levelConnExceptionTransient
	default:
		return false
	}
	banScore, ok := ps.peers[ip]
	if !ok {
		banScore = &DynamicBanScore{}
		ps.peers[ip] = banScore
	}
	score := banScore.Increase(persistent, transient)
	if score > defaultBanThreshold {
		log.WithFields(log.Fields{"module": logModule, "address": ip, "score": score, "reason": reason}).Errorf("banning and disconnecting")
		return true
	}

	if score > defaultWarnThreshold {
		log.WithFields(log.Fields{"module": logModule, "address": ip, "score": score, "reason": reason}).Warning("ban score increasing")
	}
	return false
}
