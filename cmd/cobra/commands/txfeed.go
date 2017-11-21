package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

type txFeed struct {
	Alias  string `json:"alias"`
	Filter string `json:"filter,omitempty"`
}

var createTransactionFeedCmd = &cobra.Command{
	Use:   "create-transaction-feed",
	Short: "Create a transaction feed filter",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("create-transaction-feed needs 2 args")
			return
		}

		var in txFeed
		in.Alias = args[0]
		in.Filter = args[1]

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/create-transaction-feed", &in, &response)

		jww.FEEDBACK.Printf("response: %v\n", response)
	},
}

var listTransactionFeedsCmd = &cobra.Command{
	Use:   "list-transaction-feeds",
	Short: "list all of transaction feeds",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			jww.ERROR.Println("list-transaction-feeds takes no args")
			return
		}

		var in requestQuery
		var response interface{}
		var txFeeds []txFeed

		client := mustRPCClient()
		client.Call(context.Background(), "/list-transaction-feeds", in, &response)
		if err := json.Unmarshal([]byte(response.(string)), &txFeeds); err != nil {
			jww.ERROR.Println("result not json format")
			return
		}

		for i, v := range txFeeds {
			jww.FEEDBACK.Printf("%d %v\n", i, v)
		}
	},
}

var deleteTransactionFeedCmd = &cobra.Command{
	Use:   "delete-transaction-feed",
	Short: "Delete a transaction feed filter",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("delete-transaction-feed needs 1 args")
			return
		}

		var in txFeed
		in.Alias = args[0]

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/delete-transaction-feed", &in, &response)

		jww.FEEDBACK.Printf("response: %v\n", response)
	},
}

var getTransactionFeedCmd = &cobra.Command{
	Use:   "get-transaction-feed",
	Short: "get a transaction feed by alias",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("get-transaction-feed needs 1 args")
			return
		}

		var in txFeed
		in.Alias = args[0]
		var response interface{}
		var txfeed txFeed

		client := mustRPCClient()
		client.Call(context.Background(), "/get-transaction-feed", &in, &response)
		if err := json.Unmarshal([]byte(response.(string)), &txfeed); err != nil {
			fmt.Println(err)
			jww.ERROR.Println("result not json format")
			return
		}

		jww.FEEDBACK.Printf("response: alias: %v  filter: %v\n", txfeed.Alias, txfeed.Filter)
	},
}

var updateTransactionFeedCmd = &cobra.Command{
	Use:   "update-transaction-feed",
	Short: "Update transaction feed",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("update-transaction-feed needs 2 args")
			return
		}

		var in txFeed
		in.Alias = args[0]
		in.Filter = args[1]

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/update-transaction-feed", &in, &response)

		jww.FEEDBACK.Printf("response: %v\n", response)
	},
}
