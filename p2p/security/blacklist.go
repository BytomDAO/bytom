package security

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	cfg "github.com/bytom/bytom/config"
	dbm "github.com/bytom/bytom/database/leveldb"
)

const (
	defaultBanDuration = time.Hour * 1
	blacklistKey       = "BlacklistPeers"
)

var (
	ErrConnectBannedPeer = errors.New("connect banned peer")
)

type Blacklist struct {
	peers map[string]time.Time
	db    dbm.DB

	mtx sync.Mutex
}

func NewBlacklist(config *cfg.Config) *Blacklist {
	return &Blacklist{
		peers: make(map[string]time.Time),
		db:    dbm.NewDB("blacklist", config.DBBackend, config.DBDir()),
	}
}

//AddPeer add peer to blacklist
func (bl *Blacklist) AddPeer(ip string) error {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	// delete expired banned peers
	for peer, banEnd := range bl.peers {
		if time.Now().Before(banEnd) {
			delete(bl.peers, peer)
		}
	}
	// add banned peer
	bl.peers[ip] = time.Now().Add(defaultBanDuration)
	dataJSON, err := json.Marshal(bl.peers)
	if err != nil {
		return err
	}

	bl.db.Set([]byte(blacklistKey), dataJSON)
	return nil
}

func (bl *Blacklist) delPeer(ip string) error {
	delete(bl.peers, ip)
	dataJson, err := json.Marshal(bl.peers)
	if err != nil {
		return err
	}

	bl.db.Set([]byte(blacklistKey), dataJson)
	return nil
}

func (bl *Blacklist) DoFilter(ip string, pubKey string) error {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	if banEnd, ok := bl.peers[ip]; ok {
		if time.Now().Before(banEnd) {
			return ErrConnectBannedPeer
		}

		if err := bl.delPeer(ip); err != nil {
			return err
		}
	}

	return nil
}

// LoadPeers load banned peers from db
func (bl *Blacklist) LoadPeers() error {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	if dataJSON := bl.db.Get([]byte(blacklistKey)); dataJSON != nil {
		if err := json.Unmarshal(dataJSON, &bl.peers); err != nil {
			return err
		}
	}

	return nil
}
