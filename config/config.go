package config

import (
	"encoding/hex"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
)

var (
	// CommonConfig means config object
	CommonConfig *Config
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`
	// Options for services
	P2P        *P2PConfig        `mapstructure:"p2p"`
	Wallet     *WalletConfig     `mapstructure:"wallet"`
	Auth       *RPCAuthConfig    `mapstructure:"auth"`
	Web        *WebConfig        `mapstructure:"web"`
	Websocket  *WebsocketConfig  `mapstructure:"ws"`
	Federation *FederationConfig `mapstructure:"federation"`
}

// Default configurable parameters.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
		P2P:        DefaultP2PConfig(),
		Wallet:     DefaultWalletConfig(),
		Auth:       DefaultRPCAuthConfig(),
		Web:        DefaultWebConfig(),
		Websocket:  DefaultWebsocketConfig(),
		Federation: DefaultFederationConfig(),
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
func (cfg *Config) PrivateKey() *chainkd.XPrv {
	if cfg.XPrv != nil {
		return cfg.XPrv
	}

	filePath := rootify(cfg.PrivateKeyFile, cfg.BaseConfig.RootDir)
	fildReader, err := os.Open(filePath)
	if err != nil {
		log.WithField("err", err).Panic("fail on open private key file")
	}

	defer fildReader.Close()
	buf := make([]byte, 128)
	if _, err = io.ReadFull(fildReader, buf); err != nil {
		log.WithField("err", err).Panic("fail on read private key file")
	}

	var xprv chainkd.XPrv
	if _, err := hex.Decode(xprv[:], buf); err != nil {
		log.WithField("err", err).Panic("fail on decode private key")
	}

	cfg.XPrv = &xprv
	xpub := cfg.XPrv.XPub()
	cfg.XPub = &xpub
	return cfg.XPrv
}

// -----------------------------------------------------------------------------
// BaseConfig
type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	// The alias of the node
	NodeAlias string `mapstructure:"node_alias"`

	// The ID of the network to json
	ChainID string `mapstructure:"chain_id"`

	// log level to set
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

	PrivateKeyFile string `mapstructure:"private_key_file"`
	XPrv           *chainkd.XPrv
	XPub           *chainkd.XPub

	FederationFileName string `mapstructure:"federation_file"`
}

// Default configurable base parameters.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Moniker:            "anonymous",
		ProfListenAddress:  "",
		Mining:             false,
		DBBackend:          "leveldb",
		DBPath:             "data",
		KeysPath:           "keystore",
		NodeAlias:          "",
		LogFile:            "log",
		PrivateKeyFile:     "node_key.txt",
		FederationFileName: "federation.json",
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

func (b BaseConfig) FederationFile() string {
	return rootify(b.FederationFileName, b.RootDir)
}

// P2PConfig
type P2PConfig struct {
	ListenAddress    string `mapstructure:"laddr"`
	Seeds            string `mapstructure:"seeds"`
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

// -----------------------------------------------------------------------------
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

type WebsocketConfig struct {
	MaxNumWebsockets     int `mapstructure:"max_num_websockets"`
	MaxNumConcurrentReqs int `mapstructure:"max_num_concurrent_reqs"`
}

type FederationConfig struct {
	FederationScript string         `json:"federation_script"`
	Xpubs            []chainkd.XPub `json:"xpubs"`
	Quorum           int            `json:"quorum"`
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

func DefaultWebsocketConfig() *WebsocketConfig {
	return &WebsocketConfig{
		MaxNumWebsockets:     25,
		MaxNumConcurrentReqs: 20,
	}
}

// Default configurable federation parameters.
func DefaultFederationConfig() *FederationConfig {
	return &FederationConfig{
		Xpubs: []chainkd.XPub{
			xpub("2e171e9aed46633f3560cf4d207c4edb92e5ad6b6f63daee44aa0ed4c58f76fd4d0081f225d2b119ac398749dbc7aa113603bc7710693c54852d33b6b082daab"),
			xpub("896285b552bfe0647c0effaee41e5ce98a77981396455259792300cfbc0988bfb1a723488cedf0e73c3220e273fb6843abfbee025d7401b67bf81641b208dfc1"),
			xpub("aa5cb0d5d193a141ce66dd3448e8d74d73bed1131ea05b130c14c95ad04b0295f2d4d3f421ae10a2517f7431e0eca119fea509e0650bd20b4a64b856b5473f35"),
			xpub("98e6ab8c654bb31e0c432a2c9ff13a6e3419dcb8a1df94f2839f41d79e94b6ca7a68f60b793d947195f761187b37275fbeb345041d5ea3039c5d328b63e3d489"),
		},
		Quorum: 2,
	}
}

func xpub(str string) (xpub chainkd.XPub) {
	if err := xpub.UnmarshalText([]byte(str)); err != nil {
		log.Panicf("Fail converts a string to xpub")
	}
	return xpub
}

// -----------------------------------------------------------------------------
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
