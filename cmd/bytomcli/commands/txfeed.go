package commands

import (
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var createTransactionFeedCmd = &cobra.Command{
	Use:   "create-transaction-feed <alias> <filter>",
	Short: "Create a transaction feed filter",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var in txFeed
		in.Alias = args[0]
		in.Filter = args[1]

		_, exitCode := clientCall("/create-transaction-feed", &in)
		if exitCode != Success {
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
		var in requestQuery
		var response = struct {
			Items []interface{} `json:"items"`
			Next  requestQuery  `json:"next"`
			Last  bool          `json:"last_page"`
		}{}

		idx := 0
	LOOP:
		data, exitCode := clientCall("/list-transaction-feeds", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawPage, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalParse)
		}

		if err := json.Unmarshal(rawPage, &response); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalParse)
		}

		for _, item := range response.Items {
			key := item.(string)
			jww.FEEDBACK.Printf("%d:\n%s\n\n", idx, key)
			idx++
		}
		if response.Last == false {
			in.After = response.Next.After
			goto LOOP
		}
	},
}

var deleteTransactionFeedCmd = &cobra.Command{
	Use:   "delete-transaction-feed <alias>",
	Short: "Delete a transaction feed filter",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var in txFeed
		in.Alias = args[0]

		_, exitCode := clientCall("/delete-transaction-feed", &in)
		if exitCode != Success {
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

		data, exitCode := clientCall("/get-transaction-feed", &in)
		if exitCode != Success {
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

		_, exitCode := clientCall("/update-transaction-feed", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully updated transaction feed")
	},
}
