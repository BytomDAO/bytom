package commands

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
)

func init() {
	createAccountCmd.PersistentFlags().IntVarP(&quorum, "quorom", "q", 1, "quorum must be greater than 0 and less than or equal to the number of signers")
	createAccountCmd.PersistentFlags().StringVarP(&token, "access", "a", "", "access token")
	createAccountCmd.PersistentFlags().StringVarP(&tags, "tags", "t", "", "tags")
}

var (
	quorum = 1
	token  = ""
	tags   = ""
)

var createAccountCmd = &cobra.Command{
	Use:   "create-account <alias> <xpub>",
	Short: "Create an account",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var xpub chainkd.XPub
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalExe)
		}

		var ins accountIns
		ins.RootXPubs = []chainkd.XPub{xpub}
		ins.Quorum = quorum
		ins.Alias = args[0]
		if len(tags) != 0 {
			ts := strings.Split(tags, ":")
			if len(ts) != 2 {
				jww.ERROR.Println("Invalid tags")
				os.Exit(ErrLocalExe)
			}
			ins.Tags = map[string]interface{}{ts[0]: ts[1]}
		}
		ins.AccessToken = "client"

		// account := make([]query.AnnotatedAccount, 1)
		data, exitCode := clientCall("/create-account", &ins)

		if exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println(data)
	},
}

var listAccountsCmd = &cobra.Command{
	Use:   "list-accounts",
	Short: "List the existing accounts",
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
		data, exitCode := clientCall("/list-accounts", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawPage := []byte(data[0])
		if err := json.Unmarshal(rawPage, &response); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

		for _, item := range response.Items {
			key := item.(string)
			jww.FEEDBACK.Printf("%d:\n%v\n\n", idx, key)
			idx++
		}
		if response.Last == false {
			in.After = response.Next.After
			goto LOOP
		}
	},
}
