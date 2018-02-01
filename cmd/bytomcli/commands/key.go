package commands

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/util"
)

var createKeyCmd = &cobra.Command{
	Use:   "create-key <alias> <password>",
	Short: "Create a key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var key = struct {
			Alias    string `json:"alias"`
			Password string `json:"password"`
		}{Alias: args[0], Password: args[1]}

		data, exitCode := util.ClientCall("/create-key", &key)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var deleteKeyCmd = &cobra.Command{
	Use:   "delete-key <xpub> <password>",
	Short: "Delete a key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		xpub := new(chainkd.XPub)
		if err := xpub.UnmarshalText([]byte(args[0])); err != nil {
			jww.ERROR.Println("delete-key:", err)
			os.Exit(util.ErrLocalExe)
		}

		var key = struct {
			Password string
			XPub     chainkd.XPub `json:"xpubs"`
		}{XPub: *xpub, Password: args[1]}

		if _, exitCode := util.ClientCall("/delete-key", &key); exitCode != util.Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully delete key")
	},
}

var listKeysCmd = &cobra.Command{
	Use:   "list-keys",
	Short: "List the existing keys",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/list-keys")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var exportPrivateCmd = &cobra.Command{
	Use:   "export-private-key <xpub> <password>",
	Short: "Export the private key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		type Key struct {
			Password string
			XPub     chainkd.XPub
		}
		var key Key
		xpub := new(chainkd.XPub)
		rawPub, err := hex.DecodeString(args[0])
		if err != nil {
			jww.ERROR.Println("error: export-private-key args not vaild", err)
		}
		copy(xpub[:], rawPub)

		key.XPub = *xpub
		key.Password = args[1]

		data, exitCode := util.ClientCall("/export-private-key", &key)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var importPrivateCmd = &cobra.Command{
	Use:   "import-private-key <key-alias> <private key> <index> <password> <account-alias>",
	Short: "Import the private key",
	Args:  cobra.ExactArgs(5),
	Run: func(cmd *cobra.Command, args []string) {
		var params blockchain.KeyImportParams
		params.KeyAlias = args[0]
		params.XPrv = args[1]
		params.Password = args[3]
		params.AccountAlias = args[4]
		index, err := strconv.ParseUint(args[2], 10, 64)
		if err != nil {
			jww.ERROR.Println("params index wrong")
		}
		params.Index = index

		data, exitCode := util.ClientCall("/import-private-key", &params)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var importKeyProgressCmd = &cobra.Command{
	Use:   "import-key-progress",
	Short: "Get import private key progress info",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/import-key-progress")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		fmt.Println("data:", data)
	},
}
