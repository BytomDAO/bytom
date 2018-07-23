package commands

import (
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/util"
)

func init() {
	createAccountCmd.PersistentFlags().IntVarP(&accountQuorum, "quorom", "q", 1, "quorum must be greater than 0 and less than or equal to the number of signers")
	createAccountCmd.PersistentFlags().StringVarP(&accountToken, "access", "a", "", "access token")

	listAccountsCmd.PersistentFlags().StringVar(&accountID, "id", "", "ID of account")

	listAddressesCmd.PersistentFlags().StringVar(&accountID, "id", "", "account ID")
	listAddressesCmd.PersistentFlags().StringVar(&accountAlias, "alias", "", "account alias")

	listUnspentOutputsCmd.PersistentFlags().StringVar(&outputID, "id", "", "ID of unspent output")
	listUnspentOutputsCmd.PersistentFlags().BoolVar(&smartContract, "contract", false, "list smart contract unspent outputs")
}

var (
	accountID     = ""
	accountAlias  = ""
	accountQuorum = 1
	accountToken  = ""
	outputID      = ""
	smartContract = false
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

var createAccountReceiverCmd = &cobra.Command{
	Use:   "create-account-receiver <accountAlias> [accountID]",
	Short: "Create an account receiver",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var ins = struct {
			AccountID    string `json:"account_id"`
			AccountAlias string `json:"account_alias"`
		}{AccountAlias: args[0]}

		if len(args) == 2 {
			ins.AccountID = args[1]
		}

		data, exitCode := util.ClientCall("/create-account-receiver", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var listAddressesCmd = &cobra.Command{
	Use:   "list-addresses",
	Short: "List the account addresses",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var ins = struct {
			AccountID    string `json:"account_id"`
			AccountAlias string `json:"account_alias"`
		}{AccountID: accountID, AccountAlias: accountAlias}

		data, exitCode := util.ClientCall("/list-addresses", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var validateAddressCmd = &cobra.Command{
	Use:   "validate-address <address>",
	Short: "validate the account addresses",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var ins = struct {
			Address string `json:"address"`
		}{Address: args[0]}

		data, exitCode := util.ClientCall("/validate-address", &ins)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var listPubKeysCmd = &cobra.Command{
	Use:   "list-pubkeys <accountID>",
	Short: "list the account pubkeys",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var ins = struct {
			AccountID string `json:"account_id"`
		}{AccountID: args[0]}

		data, exitCode := util.ClientCall("/list-pubkeys", &ins)
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
