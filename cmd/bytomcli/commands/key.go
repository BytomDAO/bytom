package commands

import (
	"os"

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
			Alias    string `json:"alias"`
			Password string `json:"password"`
		}{Alias: args[0], Password: args[1]}

		data, exitCode := clientCall("/create-key", &key)
		if exitCode != Success {
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
			os.Exit(ErrLocalExe)
		}

		var key = struct {
			Password string
			XPub     chainkd.XPub `json:"xpubs"`
		}{XPub: *xpub, Password: args[1]}

		if _, exitCode := clientCall("/delete-key", &key); exitCode != Success {
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
		data, exitCode := clientCall("/list-keys")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}
