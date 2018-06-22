package commands

import (
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/bytom/util"
)

var createTransactionFeedCmd = &cobra.Command{
	Use:   "create-transaction-feed <alias> <filter>",
	Short: "Create a transaction feed filter",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var in txFeed
		in.Alias = args[0]
		in.Filter = args[1]

		_, exitCode := util.ClientCall("/create-transaction-feed", &in)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully created transaction feed")
	},
}

var listTransactionFeedsCmd = &cobra.Command{
	Use:   "list-transaction-feeds",
	Short: "list all of transaction feeds",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/list-transaction-feeds")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSONList(data)
	},
}

var deleteTransactionFeedCmd = &cobra.Command{
	Use:   "delete-transaction-feed <alias>",
	Short: "Delete a transaction feed filter",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var in txFeed
		in.Alias = args[0]

		_, exitCode := util.ClientCall("/delete-transaction-feed", &in)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully deleted transaction feed")
	},
}

var getTransactionFeedCmd = &cobra.Command{
	Use:   "get-transaction-feed <alias>",
	Short: "get a transaction feed by alias",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var in txFeed
		in.Alias = args[0]

		data, exitCode := util.ClientCall("/get-transaction-feed", &in)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var updateTransactionFeedCmd = &cobra.Command{
	Use:   "update-transaction-feed <alias> <fiter>",
	Short: "Update transaction feed",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var in txFeed
		in.Alias = args[0]
		in.Filter = args[1]

		if _, exitCode := util.ClientCall("/update-transaction-feed", &in); exitCode != util.Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully updated transaction feed")
	},
}
