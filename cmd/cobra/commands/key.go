package commands

import (
	"context"
	"encoding/hex"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
)

var createKeyCmd = &cobra.Command{
	Use:   "create-key <alias> <password>",
	Short: "Create a key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var key = struct {
			Alias    string
			Password string
		}{Alias: args[0], Password: args[1]}

		data, exitCode := clientCall("/create-key", &key)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Printf("Alias: %v\nXPub: %v\nFile: %v\n", data[0], data[1], data[2])
	},
}

var deleteKeyCmd = &cobra.Command{
	Use:   "delete-key",
	Short: "Delete a key",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("delete-key needs 2 args")
			return
		}

		type Key struct {
			Password string
			XPub     chainkd.XPub `json:"xpubs"`
		}

		var key Key
		xpub := new(chainkd.XPub)
		data, err := hex.DecodeString(args[1])

		if err != nil {
			jww.ERROR.Println("delete-key %v", err)
			return
		}

		copy(xpub[:], data)
		key.Password = args[0]
		key.XPub = *xpub

		client := mustRPCClient()
		client.Call(context.Background(), "/delete-key", &key, nil)
	},
}

var listKeysCmd = &cobra.Command{
	Use:   "list-keys",
	Short: "List the existing keys",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("error: list-keys args not vaild")
			return
		}

		var in requestQuery
		in.After = args[0]
		in.PageSize, _ = strconv.Atoi(args[1])

		var response map[string][]interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/list-keys", &in, &response)

		for idx, item := range response["items"] {
			key := item.(map[string]interface{})
			jww.ERROR.Printf("%v: Alias: %v Address: %v File: %v\n", idx, key["alias"], key["address"], key["file"])
		}
	},
}
