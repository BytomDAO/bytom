package commands

import (
	"context"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Run: func(cmd *cobra.Command, args []string) {
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/gas-rate", nil, &response)
		jww.FEEDBACK.Printf("gas rate: %v\n", response)
	},
}
