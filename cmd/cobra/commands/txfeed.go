package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

type txFeed struct {
	Alias  string `json:"alias"`
	Filter string `json:"filter,omitempty"`
}

type respArrayTxFeed struct {
	Status string    `json:"status,omitempty"`
	Msg    string    `json:"msg,omitempty"`
	Data   []*txFeed `json:"data,omitempty"`
}

type respTxFeed struct {
	Status string `json:"status,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Data   txFeed `json:"data,omitempty"`
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

		var rawresp resp

		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			jww.FEEDBACK.Printf("%v\n", rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
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

		client := mustRPCClient()
		client.Call(context.Background(), "/list-transaction-feeds", in, &response)

		var rawresp respArrayTxFeed
		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			for i, v := range rawresp.Data {
				fmt.Println(i, v.Alias, v.Filter)
			}
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
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

		var rawresp resp
		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			jww.FEEDBACK.Printf("%v\n", rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
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

		client := mustRPCClient()
		client.Call(context.Background(), "/get-transaction-feed", &in, &response)

		var rawresp respTxFeed

		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			fmt.Println(rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
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

		var rawresp resp
		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			jww.FEEDBACK.Printf("%v\n", rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
	},
}
