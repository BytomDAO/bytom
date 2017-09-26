package config

import (
	"fmt"
	"path/filepath"
	"time"

	//"github.com/bytom/types"
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`

	// Options for services
	RPC       *RPCConfig       `mapstructure:"rpc"`
	P2P       *P2PConfig       `mapstructure:"p2p"`
}

func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
		RPC:        DefaultRPCConfig(),
		P2P:        DefaultP2PConfig(),
	}
}

func TestConfig() *Config {
	return &Config{
		BaseConfig: TestBaseConfig(),
		RPC:        TestRPCConfig(),
		P2P:        TestP2PConfig(),
	}
}

// Set the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	cfg.RPC.RootDir = root
	cfg.P2P.RootDir = root
	return cfg
}

//-----------------------------------------------------------------------------
// BaseConfig

type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	// The ID of the chain to join (should be signed with every transaction and vote)
	ChainID string `mapstructure:"chain_id"`

	// A JSON file containing the initial validator set and other meta data
	Genesis string `mapstructure:"genesis_file"`

	// A JSON file containing the private key to use as a validator in the consensus protocol
	PrivateKey string `mapstructure:"private_key"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// Output level for logging
	LogLevel string `mapstructure:"log_level"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	// If this node is many blocks behind the tip of the chain, FastSync
	// allows them to catchup quickly by downloading blocks in parallel
	// and verifying their commits
	FastSync bool `mapstructure:"fast_sync"`

	FilterPeers bool `mapstructure:"filter_peers"` // false

	// What indexer to use for transactions
	TxIndex string `mapstructure:"tx_index"`

	// Database backend: leveldb | memdb
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Keystore directory
	KeysPath string `mapstructure:"keys_dir"`

	// remote HSM url
	HsmUrl string `mapstructure:"hsm_url"`

	ApiAddress string `mapstructure:"api_addr"`

	Time time.Time
}

func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Genesis:           "genesis.json",
		Moniker:           "anonymous",
		LogLevel:          DefaultPackageLogLevels(),
		ProfListenAddress: "",
		FastSync:          true,
		FilterPeers:       false,
		TxIndex:           "kv",
		DBBackend:         "leveldb",
		DBPath:            "data",
		KeysPath:	   "keystore",
		HsmUrl:		   "",
	}
}

func TestBaseConfig() BaseConfig {
	conf := DefaultBaseConfig()
	conf.ChainID = "bytom_test"
	conf.FastSync = false
	conf.DBBackend = "memdb"
	return conf
}

func (b BaseConfig) GenesisFile() string {
	return rootify(b.Genesis, b.RootDir)
}

func (b BaseConfig) DBDir() string {
	return rootify(b.DBPath, b.RootDir)
}

func (b BaseConfig) KeysDir() string {
	return rootify(b.KeysPath, b.RootDir)
}


func DefaultLogLevel() string {
	return "info"
}

func DefaultPackageLogLevels() string {
	return fmt.Sprintf("state:info,*:%s", DefaultLogLevel())
}

//-----------------------------------------------------------------------------
// RPCConfig

type RPCConfig struct {
	RootDir string `mapstructure:"home"`

	// TCP or UNIX socket address for the RPC server to listen on
	ListenAddress string `mapstructure:"laddr"`

	// TCP or UNIX socket address for the gRPC server to listen on
	// NOTE: This server only supports /broadcast_tx_commit
	GRPCListenAddress string `mapstructure:"grpc_laddr"`

	// Activate unsafe RPC commands like /dial_seeds and /unsafe_flush_mempool
	Unsafe bool `mapstructure:"unsafe"`
}

func DefaultRPCConfig() *RPCConfig {
	return &RPCConfig{
		ListenAddress:     "tcp://0.0.0.0:46657",
		GRPCListenAddress: "",
		Unsafe:            false,
	}
}

func TestRPCConfig() *RPCConfig {
	conf := DefaultRPCConfig()
	conf.ListenAddress = "tcp://0.0.0.0:36657"
	conf.GRPCListenAddress = "tcp://0.0.0.0:36658"
	conf.Unsafe = true
	return conf
}

//-----------------------------------------------------------------------------
// P2PConfig

type P2PConfig struct {
	RootDir        string `mapstructure:"home"`
	ListenAddress  string `mapstructure:"laddr"`
	Seeds          string `mapstructure:"seeds"`
	SkipUPNP       bool   `mapstructure:"skip_upnp"`
	AddrBook       string `mapstructure:"addr_book_file"`
	AddrBookStrict bool   `mapstructure:"addr_book_strict"`
	PexReactor     bool   `mapstructure:"pex"`
	MaxNumPeers    int    `mapstructure:"max_num_peers"`
}

func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddress:  "tcp://0.0.0.0:46656",
		AddrBook:       "addrbook.json",
		AddrBookStrict: true,
		MaxNumPeers:    50,
	}
}

func TestP2PConfig() *P2PConfig {
	conf := DefaultP2PConfig()
	conf.ListenAddress = "tcp://0.0.0.0:36656"
	conf.SkipUPNP = true
	return conf
}

func (p *P2PConfig) AddrBookFile() string {
	return rootify(p.AddrBook, p.RootDir)
}

//-----------------------------------------------------------------------------
// Utils

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
