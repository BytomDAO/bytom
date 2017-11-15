package commands

import (
	"context"

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
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/net-listening", nil, &response)
		jww.FEEDBACK.Printf("net listening: %v\n", response)
	},
}

var peerCountCmd = &cobra.Command{
	Use:   "peer-count",
	Short: "number of peers currently connected to the client",
	Run: func(cmd *cobra.Command, args []string) {
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/peer-count", nil, &response)
		jww.FEEDBACK.Printf("peer count: %v\n", response)
	},
}
