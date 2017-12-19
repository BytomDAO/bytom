package commands

import (
	"context"
	"encoding/base64"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/env"
)

var (
	home    = blockchain.HomeDirFromEnvironment()
	coreURL = env.String("BYTOM_URL", "http://localhost:9888")
)

var blockHashCmd = &cobra.Command{
	Use:   "block-hash",
	Short: "Get the hash of most recent block",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("block-hash")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Printf("best block hash: %v\n", data)
	},
}

var blockHeightCmd = &cobra.Command{
	Use:   "block-height",
	Short: "Get the number of most recent block",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/block-height")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Printf("block height: %v\n", data)
	},
}

var getBlockByHashCmd = &cobra.Command{
	Use:   "get-block-by-hash <hash>",
	Short: "Get a whole block matching the given hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/get-block-by-hash", args[0])
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawBlock, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("%v\n", string(rawBlock))
	},
}

var getBlockHeaderByHashCmd = &cobra.Command{
	Use:   "get-block-header-by-hash",
	Short: "Get the header of a block matching the given hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/get-block-header-by-hash", args[0])
		if exitCode != Success {
			os.Exit(exitCode)
		}
		rawHeader, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("block header: %v\n", string(rawHeader))
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
	Use:   "get-block-by-height <height>",
	Short: "Get a whole block matching the given height",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		height, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			jww.ERROR.Printf("Invalid height value")
			os.Exit(ErrLocalExe)
		}

		data, exitCode := clientCall("/get-block-by-height", height)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		rawBlock, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Printf("%v\n", string(rawBlock))
	},
}

var getBlockTransactionsCountByHeightCmd = &cobra.Command{
	Use:   "get-block-transactions-count-by-height",
	Short: "Get the transactions count of a block matching the given height",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui64, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			jww.ERROR.Printf("Invalid height value")
			os.Exit(ErrLocalExe)
		}

		data, exitCode := clientCall("/get-block-transactions-count-by-height", ui64)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Printf("transactions count: %v\n", data)
	},
}
