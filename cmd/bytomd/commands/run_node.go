package commands

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/bytom/node"
)

const logModule = "cmd"

var runNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run the bytomd",
	RunE:  runNode,
}

func init() {
	runNodeCmd.Flags().String("prof_laddr", config.ProfListenAddress, "Use http to profile bytomd programs")
	runNodeCmd.Flags().Bool("mining", config.Mining, "Enable mining")

	runNodeCmd.Flags().Bool("simd.enable", config.Simd.Enable, "Enable SIMD mechan for tensority")

	runNodeCmd.Flags().Bool("auth.disable", config.Auth.Disable, "Disable rpc access authenticate")

	runNodeCmd.Flags().Bool("wallet.disable", config.Wallet.Disable, "Disable wallet")
	runNodeCmd.Flags().Bool("wallet.rescan", config.Wallet.Rescan, "Rescan wallet")
	runNodeCmd.Flags().Bool("vault_mode", config.VaultMode, "Run in the offline enviroment")
	runNodeCmd.Flags().Bool("web.closed", config.Web.Closed, "Lanch web browser or not")
	runNodeCmd.Flags().String("chain_id", config.ChainID, "Select network type")

	// log level
	runNodeCmd.Flags().String("log_level", config.LogLevel, "Select log level(debug, info, warn, error or fatal")

	// p2p flags
	runNodeCmd.Flags().String("p2p.laddr", config.P2P.ListenAddress, "Node listen address. (0.0.0.0:0 means any interface, any port)")
	runNodeCmd.Flags().String("p2p.seeds", config.P2P.Seeds, "Comma delimited host:port seed nodes")
	runNodeCmd.Flags().String("p2p.node_key", config.P2P.PrivateKey, "Node key for p2p communication")
	runNodeCmd.Flags().Bool("p2p.skip_upnp", config.P2P.SkipUPNP, "Skip UPNP configuration")
	runNodeCmd.Flags().Int("p2p.max_num_peers", config.P2P.MaxNumPeers, "Set max num peers")
	runNodeCmd.Flags().Int("p2p.handshake_timeout", config.P2P.HandshakeTimeout, "Set handshake timeout")
	runNodeCmd.Flags().Int("p2p.dial_timeout", config.P2P.DialTimeout, "Set dial timeout")
	runNodeCmd.Flags().String("p2p.proxy_address", config.P2P.ProxyAddress, "Connect via SOCKS5 proxy (eg. 127.0.0.1:1086)")
	runNodeCmd.Flags().String("p2p.proxy_username", config.P2P.ProxyUsername, "Username for proxy server")
	runNodeCmd.Flags().String("p2p.proxy_password", config.P2P.ProxyPassword, "Password for proxy server")

	// log flags
	runNodeCmd.Flags().String("log_file", config.LogFile, "Log output file")

	// websocket flags
	runNodeCmd.Flags().Int("ws.max_num_websockets", config.Websocket.MaxNumWebsockets, "Max number of websocket connections")
	runNodeCmd.Flags().Int("ws.max_num_concurrent_reqs", config.Websocket.MaxNumConcurrentReqs, "Max number of concurrent websocket requests that may be processed concurrently")

	RootCmd.AddCommand(runNodeCmd)
}

func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func runNode(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	setLogLevel(config.LogLevel)

	// Create & start node
	n := node.NewNode(config)
	if _, err := n.Start(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("failed to start node")
	}

	nodeInfo := n.NodeInfo()
	log.WithFields(log.Fields{
		"module":   logModule,
		"version":  nodeInfo.Version,
		"network":  nodeInfo.Network,
		"duration": time.Since(startTime),
	}).Info("start node complete")

	// Trap signal, run forever.
	n.RunForever()
	return nil
}
