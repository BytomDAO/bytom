package commands

import (
	"bytes"
	"context"
	"encoding/hex"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
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

var exportPrivateCmd = &cobra.Command{
	Use:   "export-private-key",
	Short: "export the private key",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			jww.ERROR.Println("error: export-private-key args not vaild export-private-key password xpub")
			return
		}

		type Key struct {
			Password string
			XPub     chainkd.XPub
		}
		var key Key
		xpub := new(chainkd.XPub)
		data, err := hex.DecodeString(args[1])
		if err != nil {
			jww.ERROR.Println("error: export-private-key args not vaild", err)
		}
		copy(xpub[:], data)

		key.Password = args[0]
		key.XPub = *xpub

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/export-private-key", &key, &response)
		res := base58.Decode(response.(string))
		if len(res) != 68 {
			jww.ERROR.Println("export private error")
			return
		}
		var hashed [32]byte
		sha3pool.Sum256(hashed[:], res[:64])
		rest := bytes.Compare(hashed[:4], res[64:])
		if rest == 0 {
			jww.FEEDBACK.Printf("priv: %v\n", response)
			return
		}
		jww.ERROR.Println("export private error")
		return
	},
}

var importPrivateCmd = &cobra.Command{
	Use:   "import-private-key",
	Short: "import the private key",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 4 {
			jww.ERROR.Println("error: import-private-key args not vaild import-private-key Alias Password Priv Index")
			return
		}

		type Key struct {
			Alias    string
			Password string
			XPrv     chainkd.XPrv
			Index    uint64
		}

		privhash := base58.Decode(args[2])
		if len(privhash) != 68 {
			jww.ERROR.Println("wif priv length error")
			return
		}
		var hashed [32]byte

		sha3pool.Sum256(hashed[:], privhash[:64])

		if res := bytes.Compare(hashed[:4], privhash[64:]); res != 0 {
			jww.ERROR.Println("wif priv hash error")
		}
		var key Key
		key.Alias = args[0]
		key.Password = args[1]
		key.Index, _ = strconv.ParseUint(args[3], 10, 64)
		copy(key.XPrv[:], privhash[:64])

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/import-private-key", &key, &response)
		jww.FEEDBACK.Printf("%v\n", response)
	},
}
