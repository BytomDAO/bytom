package commands

import (
	"context"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/env"
)

var (
	home    = blockchain.HomeDirFromEnvironment()
	coreURL = env.String("BYTOM_URL", "http://localhost:1999")
)

var bestBlockHashCmd = &cobra.Command{
	Use:   "get-best-block-hash",
	Short: "Get the most recent block hash",
	Run: func(cmd *cobra.Command, args []string) {
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-best-block-hash", nil, &response)
		jww.FEEDBACK.Printf("best block hash: %v\n", response)
	},
}

var blockCmd = &cobra.Command{
	Use:   "get-block-by-hash",
	Short: "Get a whole block matching the given hash",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println(`
get-block-by-hash args not valid.

Usage: get-block-by-hash [hash]`)
			return
		}

		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-block-by-hash", args[0], &response)
		jww.FEEDBACK.Printf("%v\n", response)
	},
}

var blockHeaderCmd = &cobra.Command{
	Use:   "get-block-header-by-hash",
	Short: "Get the header of a block matching the given hash",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println(`
get-block-header-by-hash args not valid.

Usage: get-block-header-by-hash [hash]`)
			return
		}

		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-block-header-by-hash", args[0], &response)
		jww.FEEDBACK.Printf("block header: %v\n", response)
	},
}
