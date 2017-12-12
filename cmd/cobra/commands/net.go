package commands

import (
	"context"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var netInfoCmd = &cobra.Command{
	Use:   "net-info",
	Short: "Print the summary of network",
	Run: func(cmd *cobra.Command, args []string) {
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/net-info", nil, &response)
		jww.FEEDBACK.Printf("net info: %v\n", response)
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
		res, err := strconv.ParseBool(data[0])
		if err != nil {
			jww.ERROR.Println("Fail to parse response data")
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("net listening: %v\n", res)
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
		i, err := strconv.ParseInt(data[0], 16, 64)
		if err != nil {
			jww.ERROR.Println("Fail to parse response data")
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("peer count: %v\n", i)
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
		res, err := strconv.ParseBool(data[0])
		if err != nil {
			jww.ERROR.Println("Fail to parse response data")
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("net syncing: %v\n", res)
	},
}
