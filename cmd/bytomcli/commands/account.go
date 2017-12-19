package commands

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
)

func init() {
	createAccountCmd.PersistentFlags().IntVarP(&accountQuorum, "quorom", "q", 1, "quorum must be greater than 0 and less than or equal to the number of signers")
	createAccountCmd.PersistentFlags().StringVarP(&accountToken, "access", "a", "", "access token")
	createAccountCmd.PersistentFlags().StringVarP(&accountTags, "tags", "t", "", "tags")

	updateAccountTagsCmd.PersistentFlags().StringVarP(&accountUpdateTags, "tags", "t", "", "tags to add, delete or update")
}

var (
	accountQuorum     = 1
	accountToken      = ""
	accountTags       = ""
	accountUpdateTags = ""
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
		ins.Quorum = accountQuorum
		ins.Alias = args[0]
		if len(accountTags) != 0 {
			tags := strings.Split(accountTags, ":")
			if len(tags) != 2 {
				jww.ERROR.Println("Invalid tags")
				os.Exit(ErrLocalExe)
			}
			ins.Tags = map[string]interface{}{tags[0]: tags[1]}
		}
		ins.AccessToken = accountToken

		data, exitCode := clientCall("/create-account", &ins)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawAccount, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

		jww.FEEDBACK.Println(string(rawAccount))
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

		rawPage, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

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

var deleteAccountCmd = &cobra.Command{
	Use:   "delete-account <accountID|alias>",
	Short: "Delete the existing account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, exitCode := clientCall("/delete-account", args[0]); exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully delete account")
	},
}

var updateAccountTagsCmd = &cobra.Command{
	Use:   "update-account-tags <accountID|alias>",
	Short: "Add, update or delete the account tags",
	Long: `If the tags match the pattern 'key:value', add or update them.
If the tags match the pattern 'key:', delete them.`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("tags")
	},
	Run: func(cmd *cobra.Command, args []string) {
		var updateTag = struct {
			AccountInfo string
			Tags        map[string]interface{} `json:"tags"`
		}{}

		if len(accountUpdateTags) != 0 {
			tags := strings.Split(accountUpdateTags, ":")
			if len(tags) != 2 {
				jww.ERROR.Println("Invalid tags")
				os.Exit(ErrLocalExe)
			}
			updateTag.Tags = map[string]interface{}{tags[0]: tags[1]}
		}

		updateTag.AccountInfo = args[0]
		if _, exitCode := clientCall("/update-account-tags", &updateTag); exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully update account tags")
	},
}

var createAccountReceiverCmd = &cobra.Command{
	Use:   "create-account-receiver <accountID | alias>",
	Short: "Create an account receiver control program",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		var ins = struct {
			AccountInfo string    `json:"account_info"`
			ExpiresAt   time.Time `json:"expires_at"`
		}{AccountInfo: args[0]}

		data, exitCode := clientCall("/create-account-receiver", &ins)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		rawReceiver, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}
		jww.FEEDBACK.Println(string(rawReceiver))
	},
}

var listBalances = &cobra.Command{
	Use:   "list-balances",
	Short: "List the accounts balances",
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
		data, exitCode := clientCall("/list-balances", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawPage, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

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

var listUnspentOutputs = &cobra.Command{
	Use:   "list-unspent-outputs",
	Short: "List the accounts unspent outputs",
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
		data, exitCode := clientCall("/list-unspent-outputs", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawPage, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

		if err := json.Unmarshal(rawPage, &response); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalUnwrap)
		}

		for _, item := range response.Items {
			UTXO := item.(string)
			jww.FEEDBACK.Printf("%d:\n%v\n\n", idx, UTXO)
			idx++
		}
		if response.Last == false {
			in.After = response.Next.After
			goto LOOP
		}
	},
}
