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
`

var mainNetConfigTmpl = `fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9889"
chain_id = "mainnet"
[p2p]
laddr = "tcp://0.0.0.0:46660"
seeds = "52.83.251.197:46660"
`

var testNetConfigTmpl = `fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9890"
chain_id = "wisdom"
[p2p]
laddr = "tcp://0.0.0.0:46659"
seeds = "52.83.251.197:46659"
`

var soloNetConfigTmpl = `fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9891"
chain_id = "solonet"
[p2p]
laddr = "tcp://0.0.0.0:46661"
seeds = ""
`

// Select network seeds to merge a new string.
func selectNetwork(network string) string {
	switch network {
	case "mainnet":
		return defaultConfigTmpl + mainNetConfigTmpl
	case "testnet":
		return defaultConfigTmpl + testNetConfigTmpl
	default:
		return defaultConfigTmpl + soloNetConfigTmpl
	}
}
