package config

import (
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/crypto/ed25519"
)

var (
	// CommonConfig means config object
	CommonConfig *Config
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`
	// Options for services
	P2P       *P2PConfig       `mapstructure:"p2p"`
	Wallet    *WalletConfig    `mapstructure:"wallet"`
	Auth      *RPCAuthConfig   `mapstructure:"auth"`
	Web       *WebConfig       `mapstructure:"web"`
	Simd      *SimdConfig      `mapstructure:"simd"`
	Websocket *WebsocketConfig `mapstructure:"ws"`
}

// Default configurable parameters.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
		P2P:        DefaultP2PConfig(),
		Wallet:     DefaultWalletConfig(),
		Auth:       DefaultRPCAuthConfig(),
		Web:        DefaultWebConfig(),
		Simd:       DefaultSimdConfig(),
		Websocket:  DefaultWebsocketConfig(),
	}
}

// Set the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	return cfg
}

// NodeKey retrieves the currently configured private key of the node, checking
// first any manually set key, falling back to the one found in the configured
// data folder. If no key can be found, a new one is generated.
func (cfg *Config) NodeKey() (string, error) {
	// Use any specifically configured key.
	if cfg.P2P.PrivateKey != "" {
		return cfg.P2P.PrivateKey, nil
	}

	keyFile := rootify(cfg.P2P.NodeKeyFile, cfg.BaseConfig.RootDir)
	buf := make([]byte, ed25519.PrivateKeySize*2)
	fd, err := os.Open(keyFile)
	defer fd.Close()
	if err == nil {
		if _, err = io.ReadFull(fd, buf); err == nil {
			return string(buf), nil
		}
	}

	log.WithField("err", err).Warning("key file access failed")
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", err
	}

	if err = ioutil.WriteFile(keyFile, []byte(privKey.String()), 0600); err != nil {
		return "", err
	}
	return privKey.String(), nil
}

//-----------------------------------------------------------------------------
// BaseConfig
type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	//The alias of the node
	NodeAlias string `mapstructure:"node_alias"`

	//The ID of the network to json
	ChainID string `mapstructure:"chain_id"`

	//log level to set
	LogLevel string `mapstructure:"log_level"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	Mining bool `mapstructure:"mining"`

	// Database backend: leveldb | memdb
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Keystore directory
	KeysPath string `mapstructure:"keys_dir"`

	ApiAddress string `mapstructure:"api_addr"`

	VaultMode bool `mapstructure:"vault_mode"`

	// log file name
	LogFile string `mapstructure:"log_file"`
}

// Default configurable base parameters.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Moniker:           "anonymous",
		ProfListenAddress: "",
		Mining:            false,
		DBBackend:         "leveldb",
		DBPath:            "data",
		KeysPath:          "keystore",
		NodeAlias:         "",
		LogFile:           "log",
	}
}

func (b BaseConfig) DBDir() string {
	return rootify(b.DBPath, b.RootDir)
}

func (b BaseConfig) LogDir() string {
	return rootify(b.LogFile, b.RootDir)
}

func (b BaseConfig) KeysDir() string {
	return rootify(b.KeysPath, b.RootDir)
}

// P2PConfig
type P2PConfig struct {
	ListenAddress    string `mapstructure:"laddr"`
	Seeds            string `mapstructure:"seeds"`
	PrivateKey       string `mapstructure:"node_key"`
	NodeKeyFile      string `mapstructure:"node_key_file"`
	SkipUPNP         bool   `mapstructure:"skip_upnp"`
	LANDiscover      bool   `mapstructure:"lan_discoverable"`
	MaxNumPeers      int    `mapstructure:"max_num_peers"`
	HandshakeTimeout int    `mapstructure:"handshake_timeout"`
	DialTimeout      int    `mapstructure:"dial_timeout"`
	ProxyAddress     string `mapstructure:"proxy_address"`
	ProxyUsername    string `mapstructure:"proxy_username"`
	ProxyPassword    string `mapstructure:"proxy_password"`
	KeepDial         string `mapstructure:"keep_dial"`
}

// Default configurable p2p parameters.
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddress:    "tcp://0.0.0.0:46656",
		NodeKeyFile:      "nodekey",
		SkipUPNP:         false,
		LANDiscover:      true,
		MaxNumPeers:      50,
		HandshakeTimeout: 30,
		DialTimeout:      3,
		ProxyAddress:     "",
		ProxyUsername:    "",
		ProxyPassword:    "",
	}
}

//-----------------------------------------------------------------------------
type WalletConfig struct {
	Disable  bool   `mapstructure:"disable"`
	Rescan   bool   `mapstructure:"rescan"`
	TxIndex  bool   `mapstructure:"txindex"`
	MaxTxFee uint64 `mapstructure:"max_tx_fee"`
}

type RPCAuthConfig struct {
	Disable bool `mapstructure:"disable"`
}

type WebConfig struct {
	Closed bool `mapstructure:"closed"`
}

type SimdConfig struct {
	Enable bool `mapstructure:"enable"`
}

type WebsocketConfig struct {
	MaxNumWebsockets     int `mapstructure:"max_num_websockets"`
	MaxNumConcurrentReqs int `mapstructure:"max_num_concurrent_reqs"`
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
		Disable:  false,
		Rescan:   false,
		TxIndex:  false,
		MaxTxFee: uint64(1000000000),
	}
}

// Default configurable web parameters.
func DefaultSimdConfig() *SimdConfig {
	return &SimdConfig{
		Enable: false,
	}
}

func DefaultWebsocketConfig() *WebsocketConfig {
	return &WebsocketConfig{
		MaxNumWebsockets:     25,
		MaxNumConcurrentReqs: 20,
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
		// In order to be compatible with old data path,
		// copy the data from the old path to the new path
		oldPath := filepath.Join(home, "Library", "Bytom")
		newPath := filepath.Join(home, "Library", "Application Support", "Bytom")
		if !isFolderNotExists(oldPath) && isFolderNotExists(newPath) {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Errorf("DefaultDataDir: %v", err)
				return oldPath
			}
		}
		return newPath
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Bytom")
	default:
		return filepath.Join(home, ".bytom")
	}
}

func isFolderNotExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
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
