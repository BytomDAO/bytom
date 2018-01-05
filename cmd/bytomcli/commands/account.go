package commands

import (
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

	listAccountsCmd.PersistentFlags().StringVar(&accountID, "id", "", "ID of account")

	listUnspentOutputsCmd.PersistentFlags().StringVar(&outputID, "id", "", "ID of unspent output")
}

var (
	accountID         = ""
	accountQuorum     = 1
	accountToken      = ""
	accountTags       = ""
	accountUpdateTags = ""
	outputID          = ""
)

var createAccountCmd = &cobra.Command{
	Use:   "create-account <alias> <xpub(s)>",
	Short: "Create an account",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ins := accountIns{}

		for _, x := range args[1:] {
			xpub := chainkd.XPub{}
			if err := xpub.UnmarshalText([]byte(x)); err != nil {
				jww.ERROR.Println(err)
				os.Exit(ErrLocalExe)
			}
			ins.RootXPubs = append(ins.RootXPubs, xpub)
		}

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

		printJSON(data)
	},
}

var listAccountsCmd = &cobra.Command{
	Use:   "list-accounts",
	Short: "List the existing accounts",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		filter := struct {
			ID string `json:"id"`
		}{ID: accountID}

		data, exitCode := clientCall("/list-accounts", &filter)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var deleteAccountCmd = &cobra.Command{
	Use:   "delete-account <accountID|alias>",
	Short: "Delete the existing account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		accountInfo := &struct {
			AccountInfo string `json:"account_info"`
		}{AccountInfo: args[0]}

		if _, exitCode := clientCall("/delete-account", accountInfo); exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully delete account")
	},
}

var updateAccountTagsCmd = &cobra.Command{
	Use:   "update-account-tags <accountID|alias>",
	Short: "Update the account tags",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("tags")
	},
	Run: func(cmd *cobra.Command, args []string) {
		var updateTag = struct {
			AccountInfo string                 `json:"account_info"`
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
			ExpiresAt   time.Time `json:"expires_at,omitempty"`
		}{AccountInfo: args[0]}

		data, exitCode := clientCall("/create-account-receiver", &ins)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var createAccountAddressCmd = &cobra.Command{
	Use:   "create-account-address <accountID | alias>",
	Short: "Create an account address",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var ins = struct {
			AccountInfo string    `json:"account_info"`
			ExpiresAt   time.Time `json:"expires_at,omitempty"`
		}{AccountInfo: args[0]}

		data, exitCode := clientCall("/create-account-address", &ins)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var listBalancesCmd = &cobra.Command{
	Use:   "list-balances",
	Short: "List the accounts balances",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/list-balances")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var listUnspentOutputsCmd = &cobra.Command{
	Use:   "list-unspent-outputs",
	Short: "List the accounts unspent outputs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		filter := struct {
			ID string `json:"id"`
		}{ID: outputID}

		data, exitCode := clientCall("/list-unspent-outputs", &filter)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}
