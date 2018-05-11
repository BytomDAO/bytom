package config

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`
	// Options for services
	P2P    *P2PConfig     `mapstructure:"p2p"`
	Wallet *WalletConfig  `mapstructure:"wallet"`
	Auth   *RPCAuthConfig `mapstructure:"auth"`
	Web    *WebConfig     `mapstructure:"web"`
}

// Default configurable parameters.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
		P2P:        DefaultP2PConfig(),
		Wallet:     DefaultWalletConfig(),
		Auth:       DefaultRPCAuthConfig(),
		Web:        DefaultWebConfig(),
	}
}

// Set the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	cfg.P2P.RootDir = root
	return cfg
}

//-----------------------------------------------------------------------------
// BaseConfig
type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	//The ID of the network to json
	ChainID string `mapstructure:"chain_id"`

	// A JSON file containing the private key to use as a validator in the consensus protocol
	PrivateKey string `mapstructure:"private_key"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	// If this node is many blocks behind the tip of the chain, FastSync
	// allows them to catchup quickly by downloading blocks in parallel
	// and verifying their commits
	FastSync bool `mapstructure:"fast_sync"`

	Mining bool `mapstructure:"mining"`

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

	VaultMode bool `mapstructure:"vault_mode"`

	Time time.Time

	// log file name
	LogFile string `mapstructure:"log_file"`
}

// Default configurable base parameters.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Moniker:           "anonymous",
		ProfListenAddress: "",
		FastSync:          true,
		FilterPeers:       false,
		Mining:            false,
		TxIndex:           "kv",
		DBBackend:         "leveldb",
		DBPath:            "data",
		KeysPath:          "keystore",
		HsmUrl:            "",
	}
}

func (b BaseConfig) DBDir() string {
	return rootify(b.DBPath, b.RootDir)
}

func (b BaseConfig) KeysDir() string {
	return rootify(b.KeysPath, b.RootDir)
}

// P2PConfig
type P2PConfig struct {
	RootDir          string `mapstructure:"home"`
	ListenAddress    string `mapstructure:"laddr"`
	Seeds            string `mapstructure:"seeds"`
	SkipUPNP         bool   `mapstructure:"skip_upnp"`
	AddrBook         string `mapstructure:"addr_book_file"`
	AddrBookStrict   bool   `mapstructure:"addr_book_strict"`
	PexReactor       bool   `mapstructure:"pex"`
	MaxNumPeers      int    `mapstructure:"max_num_peers"`
	HandshakeTimeout int    `mapstructure:"handshake_timeout"`
	DialTimeout      int    `mapstructure:"dial_timeout"`
}

// Default configurable p2p parameters.
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddress:    "tcp://0.0.0.0:46656",
		AddrBook:         "addrbook.json",
		AddrBookStrict:   true,
		SkipUPNP:         false,
		MaxNumPeers:      50,
		HandshakeTimeout: 30,
		DialTimeout:      3,
		PexReactor:       true,
	}
}

func (p *P2PConfig) AddrBookFile() string {
	return rootify(p.AddrBook, p.RootDir)
}

//-----------------------------------------------------------------------------
type WalletConfig struct {
	Disable bool `mapstructure:"disable"`
	Rescan  bool `mapstructure:"rescan"`
}

type RPCAuthConfig struct {
	Disable bool `mapstructure:"disable"`
}

type WebConfig struct {
	Closed bool `mapstructure:"closed"`
}

// Default configurable rpc's auth parameters.
func DefaultRPCAuthConfig() *RPCAuthConfig {
	return &RPCAuthConfig{
		Disable: false,
	}
}

// Default configurable web parameters.
func DefaultWebConfig() *WebConfig {
	return &WebConfig{
		Closed: false,
	}
}

// Default configurable wallet parameters.
func DefaultWalletConfig() *WalletConfig {
	return &WalletConfig{
		Disable: false,
		Rescan:  false,
	}
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

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home == "" {
		return "./.bytom"
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Bytom")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Bytom")
	default:
		return filepath.Join(home, ".bytom")
	}
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
