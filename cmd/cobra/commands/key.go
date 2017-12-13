package commands

import (
	"encoding/json"
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
		jww.FEEDBACK.Println("Success delete key")
	},
}

var listKeysCmd = &cobra.Command{
	Use:   "list-keys",
	Short: "List the existing keys",
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
		data, exitCode := clientCall("/list-keys", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawPage := []byte(data[0])
		if err := json.Unmarshal(rawPage, &response); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

		for _, item := range response.Items {
			key := item.(map[string]interface{})
			jww.FEEDBACK.Printf("%v:\nAlias: %v\nXpub: %v\nFile: %v\n\n", idx, key["alias"], key["xpub"], key["file"])
			idx++
		}
		if response.Last == false {
			in.After = response.Next.After
			goto LOOP
		}
	},
}
