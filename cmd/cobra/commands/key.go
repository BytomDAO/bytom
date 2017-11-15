package commands

import (
	"context"
	"encoding/hex"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/encoding/json"
)

var createKeyCmd = &cobra.Command{
	Use:   "create-key",
	Short: "Create a key",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("create-key needs 2 args")
			return
		}

		type Key struct {
			Alias    string
			Password string
		}

		var key Key
		var response map[string]interface{}

		key.Alias = args[0]
		key.Password = args[1]

		client := mustRPCClient()
		client.Call(context.Background(), "/create-key", &key, &response)

		jww.FEEDBACK.Printf("Alias: %v,  XPub: %v, File: %v\n", response["alias"], response["xpub"], response["file"])
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

var listKeyCmd = &cobra.Command{
	Use:   "list-key",
	Short: "List the existing key",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("error: listKeys args not vaild")
			return
		}

		type requestQuery struct {
			Filter       string        `json:"filter,omitempty"`
			FilterParams []interface{} `json:"filter_params,omitempty"`
			SumBy        []string      `json:"sum_by,omitempty"`
			PageSize     int           `json:"page_size"`
			AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
			Timeout      json.Duration `json:"timeout"`
			After        string        `json:"after"`
			StartTimeMS  uint64        `json:"start_time,omitempty"`
			EndTimeMS    uint64        `json:"end_time,omitempty"`
			TimestampMS  uint64        `json:"timestamp,omitempty"`
			Type         string        `json:"type"`
			Aliases      []string      `json:"aliases,omitempty"`
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
