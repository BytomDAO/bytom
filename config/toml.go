package config

import (
	"os"
	"os/exec"
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

	cmn.EnsureDir(rootDir+"/key", 0700)
	if err := os.Chdir(rootDir + "/key"); err != nil {
		panic(err)
	}

	cmd := exec.Command("/bin/bash", "-c", `go run $GOROOT/src/crypto/tls/generate_cert.go --host="localhost"`)
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

var defaultConfigTmpl = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml
fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9888"
node_alias = ""
`

var mainNetConfigTmpl = `chain_id = "mainnet"
[p2p]
laddr = "tcp://0.0.0.0:46657"
seeds = "45.79.213.28:46657,198.74.61.131:46657,212.111.41.245:46657,47.100.214.154:46657,47.100.109.199:46657,47.100.105.165:46657"
`

var testNetConfigTmpl = `chain_id = "wisdom"
[p2p]
laddr = "tcp://0.0.0.0:46656"
seeds = "52.83.107.224:46656,52.83.251.197:46656"
`

var soloNetConfigTmpl = `chain_id = "solonet"
[p2p]
laddr = "tcp://0.0.0.0:46658"
seeds = ""
`

var httpsConfigTmpl = `
[https]
enable_tls = true
cert_file = "key/cert.pem"
key_file = "key/key.pem"
`

// Select network seeds to merge a new string.
func selectNetwork(network string) string {
	switch network {
	case "mainnet":
		return defaultConfigTmpl + mainNetConfigTmpl + httpsConfigTmpl
	case "testnet":
		return defaultConfigTmpl + testNetConfigTmpl + httpsConfigTmpl
	default:
		return defaultConfigTmpl + soloNetConfigTmpl + httpsConfigTmpl
	}
}
