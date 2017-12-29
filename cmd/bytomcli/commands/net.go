package commands

import (
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var netInfoCmd = &cobra.Command{
	Use:   "net-info",
	Short: "Print the summary of network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/net-info")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
var netListeningCmd = &cobra.Command{
	Use:   "net-listening",
	Short: "If client is actively listening for network connections",
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/net-info")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		net := data.(map[string]interface{})
		jww.FEEDBACK.Printf("net listening: %v\n", net["listening"])
	},
}

var peerCountCmd = &cobra.Command{
	Use:   "peer-count",
	Short: "Number of peers currently connected to the client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/net-info")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		net := data.(map[string]interface{})
		jww.FEEDBACK.Printf("peer count: %v\n", net["peer_count"])
	},
}

var netSyncingCmd = &cobra.Command{
	Use:   "net-syncing",
	Short: "If the network is still syncing",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/net-info")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		net := data.(map[string]interface{})
		jww.FEEDBACK.Printf("net syncing: %v\n", net["syncing"])
	},
}
