package config

import (
	"path"

	cmn "github.com/tendermint/tmlibs/common"
)

/****** these are for production settings ***********/
func EnsureRoot(rootDir string, network string) {
	cmn.EnsureDir(rootDir, 0700)
	cmn.EnsureDir(rootDir+"/data", 0700)

	configFilePath := path.Join(rootDir, "config.toml")

	// Write default config file if missing.
	if !cmn.FileExists(configFilePath) {
		cmn.MustWriteFile(configFilePath, []byte(selectNetwork(network)), 0644)
	}
}

var defaultConfigTmpl = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml
fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9888"
`

var mainNetConfigTmpl = `chain_id = "mainnet"
[p2p]
laddr = "tcp://0.0.0.0:46656"
seeds = ""
`

var testNetConfigTmpl = `chain_id = "testnet"
[p2p]
laddr = "tcp://0.0.0.0:46656"
seeds = "172.104.224.219:46656,198.74.61.131:46656,212.111.41.245:46656"
`

// Select network seeds to merge a new string.
func selectNetwork(network string) string {
	if network == "testnet" {
		return defaultConfigTmpl + testNetConfigTmpl
	} else {
		return defaultConfigTmpl + mainNetConfigTmpl
	}
}
