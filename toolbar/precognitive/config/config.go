package config

import (
	"encoding/json"
	"os"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/toolbar/common"
	log "github.com/sirupsen/logrus"
)

func NewConfig() *Config {
	if len(os.Args) <= 1 {
		log.Fatal("Please setup the config file path")
	}

	return NewConfigWithPath(os.Args[1])
}

func NewConfigWithPath(path string) *Config {
	configFile, err := os.Open(path)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "file_path": os.Args[1]}).Fatal("fail to open config file")
	}
	defer configFile.Close()

	cfg := &Config{}
	if err := json.NewDecoder(configFile).Decode(cfg); err != nil {
		log.WithField("err", err).Fatal("fail to decode config file")
	}

	return cfg
}

type Config struct {
	NetworkID        uint64             `json:"network_id"`
	MySQLConfig      common.MySQLConfig `json:"mysql"`
	CheckFreqMinutes uint64             `json:"check_frequency_minutes"`
	Policy           Policy             `json:"policy"`
	Nodes            []Node             `json:"seeds"`
	API              API                `json:"api"`
}

type Policy struct {
	Confirmations uint64 `json:"confirmations"`
	RequiredRttMS uint64 `json:"required_rtt_ms"`
}

type Node struct {
	XPub      *chainkd.XPub `json:"xpub"`
	PublicKey string        `json:"public_key"`
	IP        string        `json:"ip"`
	Port      uint16        `json:"port"`
}

type API struct {
	ListeningPort uint64 `json:"listening_port"`
	AccessToken   string `json:"access_token"`
	IsReleaseMode bool   `json:"is_release_mode"`
}
