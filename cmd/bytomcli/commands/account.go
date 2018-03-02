package commands

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/util"
)

func init() {
	createAccountCmd.PersistentFlags().IntVarP(&accountQuorum, "quorom", "q", 1, "quorum must be greater than 0 and less than or equal to the number of signers")
	createAccountCmd.PersistentFlags().StringVarP(&accountToken, "access", "a", "", "access token")
	createAccountCmd.PersistentFlags().StringVarP(&accountTags, "tags", "t", "", "tags")

	updateAccountTagsCmd.PersistentFlags().StringVarP(&accountUpdateTags, "tags", "t", "", "tags to add, delete or update")

	listAccountsCmd.PersistentFlags().StringVar(&accountID, "id", "", "ID of account")

	listUnspentOutputsCmd.PersistentFlags().StringVar(&outputID, "id", "", "ID of unspent output")
	listUnspentOutputsCmd.PersistentFlags().BoolVar(&smartContract, "contract", false, "list smart contract unspent outputs")
}

var (
	accountID         = ""
	accountQuorum     = 1
	accountToken      = ""
	accountTags       = ""
	accountUpdateTags = ""
	outputID          = ""
	smartContract     = false
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
				os.Exit(util.ErrLocalExe)
			}
			ins.RootXPubs = append(ins.RootXPubs, xpub)
		}

		ins.Quorum = accountQuorum
		ins.Alias = args[0]
		if len(accountTags) != 0 {
			tags := strings.Split(accountTags, ":")
			if len(tags) != 2 {
				jww.ERROR.Println("Invalid tags")
				os.Exit(util.ErrLocalExe)
			}
			ins.Tags = map[string]interface{}{tags[0]: tags[1]}
		}

		ins.AccessToken = accountToken

		data, exitCode := util.ClientCall("/create-account", &ins)
		if exitCode != util.Success {
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

		data, exitCode := util.ClientCall("/list-accounts", &filter)
		if exitCode != util.Success {
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

		if _, exitCode := util.ClientCall("/delete-account", accountInfo); exitCode != util.Success {
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
				os.Exit(util.ErrLocalExe)
			}
			updateTag.Tags = map[string]interface{}{tags[0]: tags[1]}
		}

		updateTag.AccountInfo = args[0]

		if _, exitCode := util.ClientCall("/update-account-tags", &updateTag); exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Successfully update account tags")
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

		data, exitCode := util.ClientCall("/create-account-address", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var createAccountPubkeyCmd = &cobra.Command{
	Use:   "create-account-pubkey <accountID | alias>",
	Short: "Create an account pubkey",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var ins = struct {
			AccountInfo string `json:"account_info"`
		}{AccountInfo: args[0]}

		data, exitCode := util.ClientCall("/create-account-pubkey", &ins)
		if exitCode != util.Success {
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
		data, exitCode := util.ClientCall("/list-balances")
		if exitCode != util.Success {
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
			ID            string `json:"id"`
			SmartContract bool   `json:"smart_contract"`
		}{ID: outputID, SmartContract: smartContract}

		data, exitCode := util.ClientCall("/list-unspent-outputs", &filter)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}
