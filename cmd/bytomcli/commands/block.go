package commands

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/bytom/util"
)

var blockHashCmd = &cobra.Command{
	Use:   "block-hash",
	Short: "Get the hash of most recent block",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("block-hash")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var blockHeightCmd = &cobra.Command{
	Use:   "block-height",
	Short: "Get the number of most recent block",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/block-height")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var getBlockByHashCmd = &cobra.Command{
	Use:   "get-block-by-hash <hash>",
	Short: "Get a whole block matching the given hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/get-block-by-hash", args[0])
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var getBlockHeaderByHashCmd = &cobra.Command{
	Use:   "get-block-header-by-hash",
	Short: "Get the header of a block matching the given hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/get-block-header-by-hash", args[0])
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var getBlockTransactionsCountByHashCmd = &cobra.Command{
	Use:   "get-block-transactions-count-by-hash",
	Short: "Get the transactions count of a block matching the given hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/get-block-transactions-count-by-hash", args[0])
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
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
			os.Exit(util.ErrLocalExe)
		}

		data, exitCode := util.ClientCall("/get-block-by-height", height)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
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
			os.Exit(util.ErrLocalExe)
		}

		data, exitCode := util.ClientCall("/get-block-transactions-count-by-height", ui64)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}
