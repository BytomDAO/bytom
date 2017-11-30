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

[p2p]
laddr = "tcp://0.0.0.0:46656"
`

var testnetSeeds = `
seeds = "139.162.105.40:46656,139.162.88.74:46656,47.96.42.1:46656,45.79.213.28:46656,212.111.41.245:46656"
`
var mainnetSeeds = `seeds = ""`

// Select network seeds to merge a new string.
func selectNetwork(network string) string {
	if network == "testnet" {
		return defaultConfigTmpl + testnetSeeds
	} else {
		return defaultConfigTmpl + mainnetSeeds
	}
}
