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

		resultMap, ok := data.(map[string]interface{})
		if ok != true {
			jww.ERROR.Println("invalid type assertion")
			os.Exit(ErrLocalParse)
		}
		jww.FEEDBACK.Printf("listening:%v\nsyncing:%v\npeer_count:%v\n",
			resultMap["listening"], resultMap["syncing"], resultMap["peer_count"])
	},
}
var netListeningCmd = &cobra.Command{
	Use:   "net-listening",
	Short: "If client is actively listening for network connections",
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/net-listening")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Printf("net listening: %v\n", data)
	},
}

var peerCountCmd = &cobra.Command{
	Use:   "peer-count",
	Short: "Number of peers currently connected to the client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/peer-count")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Printf("peer count: %v\n", data)
	},
}

var netSyncingCmd = &cobra.Command{
	Use:   "net-syncing",
	Short: "If the network is still syncing",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/net-syncing")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Printf("net syncing: %v\n", data)
	},
}
