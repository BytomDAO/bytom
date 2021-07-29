package config

import (
	"encoding/json"
	"os"

	"github.com/bytom/bytom/toolbar/common"
)

type Config struct {
	NodeIP      string             `json:"node_ip"`
	ChainID     string             `json:"chain_id"`
	MySQLConfig common.MySQLConfig `json:"mysql"`
	RewardConf  *RewardConfig      `json:"reward_config"`
}

type RewardConfig struct {
	XPub          string `json:"xpub"`
	AccountID     string `json:"account_id"`
	Password      string `json:"password"`
	MiningAddress string `json:"mining_address"`
	RewardRatio   uint64 `json:"reward_ratio"`
}

func LoadConfigFile(configFile string, config *Config) error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}
