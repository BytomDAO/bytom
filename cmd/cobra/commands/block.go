package commands

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/env"
)

var (
	home    = blockchain.HomeDirFromEnvironment()
	coreURL = env.String("BYTOM_URL", "http://localhost:1999")
)

var getBestBlockHashCmd = &cobra.Command{
	Use:   "get-best-block-hash",
	Short: "Get the most recent block hash",
	Run: func(cmd *cobra.Command, args []string) {
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-best-block-hash", nil, &response)
		jww.FEEDBACK.Printf("best block hash: %v\n", response)
	},
}

var getBlockByHashCmd = &cobra.Command{
	Use:   "get-block-by-hash",
	Short: "Get a whole block matching the given hash",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("get-block-by-hash args not valid\nUsage: get-block-by-hash [hash]")
			return
		}

		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-block-by-hash", args[0], &response)
		jww.FEEDBACK.Printf("%v\n", response)
	},
}

var getBlockHeaderByHashCmd = &cobra.Command{
	Use:   "get-block-header-by-hash",
	Short: "Get the header of a block matching the given hash",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("get-block-header-by-hash args not valid\nUsage: get-block-header-by-hash [hash]")
			return
		}

		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-block-header-by-hash", args[0], &response)
		jww.FEEDBACK.Printf("block header: %v\n", response)
	},
}

var getBlockTransactionsCountByHashCmd = &cobra.Command{
	Use:   "get-block-transactions-count-by-hash",
	Short: "Get the transactions count of a block matching the given hash",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("get-block-header-by-hash args not valid\nUsage: get-block-transactions-count-by-hash [hash]")
			return
		}

		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-block-transactions-count-by-hash", args[0], &response)
		jww.FEEDBACK.Printf("transactions count: %v\n", response)
	},
}

var getBlockByHeightCmd = &cobra.Command{
	Use:   "get-block-by-height",
	Short: "Get a whole block matching the given height",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("get-block-by-height args not valid\nUsage: get-block-by-height [height]")
			return
		}
		ui64, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			jww.ERROR.Printf("Invalid height value")
			return
		}
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/get-block-by-height", ui64, &response)
		jww.FEEDBACK.Printf("%v\n", response)
	},
}
