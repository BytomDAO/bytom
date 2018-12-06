package commands

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/bytom/util"
)

// bytomcli usage template
var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:
    {{range .Commands}}{{if (and .IsAvailableCommand (.Name | WalletDisable))}}
    {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

  available with wallet enable:
    {{range .Commands}}{{if (and .IsAvailableCommand (.Name | WalletEnable))}}
    {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// commandError is an error used to signal different error situations in command handling.
type commandError struct {
	s         string
	userError bool
}

func (c commandError) Error() string {
	return c.s
}

func (c commandError) isUserError() bool {
	return c.userError
}

func newUserError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: true}
}

func newSystemError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: false}
}

func newSystemErrorF(format string, a ...interface{}) commandError {
	return commandError{s: fmt.Sprintf(format, a...), userError: false}
}

// Catch some of the obvious user errors from Cobra.
// We don't want to show the usage message for every error.
// The below may be to generic. Time will show.
var userErrorRegexp = regexp.MustCompile("argument|flag|shorthand")

func isUserError(err error) bool {
	if cErr, ok := err.(commandError); ok && cErr.isUserError() {
		return true
	}

	return userErrorRegexp.MatchString(err.Error())
}

// BytomcliCmd is Bytomcli's root command.
// Every other command attached to BytomcliCmd is a child command to it.
var BytomcliCmd = &cobra.Command{
	Use:   "bytomcli",
	Short: "Bytomcli is a commond line client for bytom core (a.k.a. bytomd)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.SetUsageTemplate(usageTemplate)
			cmd.Usage()
		}
	},
}

// Execute adds all child commands to the root command BytomcliCmd and sets flags appropriately.
func Execute() {

	AddCommands()
	AddTemplateFunc()

	if _, err := BytomcliCmd.ExecuteC(); err != nil {
		os.Exit(util.ErrLocalExe)
	}
}

// AddCommands adds child commands to the root command BytomcliCmd.
func AddCommands() {
	BytomcliCmd.AddCommand(createAccessTokenCmd)
	BytomcliCmd.AddCommand(listAccessTokenCmd)
	BytomcliCmd.AddCommand(deleteAccessTokenCmd)
	BytomcliCmd.AddCommand(checkAccessTokenCmd)

	BytomcliCmd.AddCommand(createAccountCmd)
	BytomcliCmd.AddCommand(deleteAccountCmd)
	BytomcliCmd.AddCommand(listAccountsCmd)
	BytomcliCmd.AddCommand(updateAccountAliasCmd)
	BytomcliCmd.AddCommand(createAccountReceiverCmd)
	BytomcliCmd.AddCommand(listAddressesCmd)
	BytomcliCmd.AddCommand(validateAddressCmd)
	BytomcliCmd.AddCommand(listPubKeysCmd)

	BytomcliCmd.AddCommand(createAssetCmd)
	BytomcliCmd.AddCommand(getAssetCmd)
	BytomcliCmd.AddCommand(listAssetsCmd)
	BytomcliCmd.AddCommand(updateAssetAliasCmd)

	BytomcliCmd.AddCommand(getTransactionCmd)
	BytomcliCmd.AddCommand(listTransactionsCmd)

	BytomcliCmd.AddCommand(getUnconfirmedTransactionCmd)
	BytomcliCmd.AddCommand(listUnconfirmedTransactionsCmd)
	BytomcliCmd.AddCommand(decodeRawTransactionCmd)

	BytomcliCmd.AddCommand(listUnspentOutputsCmd)
	BytomcliCmd.AddCommand(listBalancesCmd)

	BytomcliCmd.AddCommand(rescanWalletCmd)
	BytomcliCmd.AddCommand(walletInfoCmd)

	BytomcliCmd.AddCommand(buildTransactionCmd)
	BytomcliCmd.AddCommand(signTransactionCmd)
	BytomcliCmd.AddCommand(submitTransactionCmd)
	BytomcliCmd.AddCommand(estimateTransactionGasCmd)

	BytomcliCmd.AddCommand(getBlockCountCmd)
	BytomcliCmd.AddCommand(getBlockHashCmd)
	BytomcliCmd.AddCommand(getBlockCmd)
	BytomcliCmd.AddCommand(getBlockHeaderCmd)
	BytomcliCmd.AddCommand(getDifficultyCmd)
	BytomcliCmd.AddCommand(getHashRateCmd)

	BytomcliCmd.AddCommand(createKeyCmd)
	BytomcliCmd.AddCommand(deleteKeyCmd)
	BytomcliCmd.AddCommand(listKeysCmd)
	BytomcliCmd.AddCommand(updateKeyAliasCmd)
	BytomcliCmd.AddCommand(resetKeyPwdCmd)
	BytomcliCmd.AddCommand(checkKeyPwdCmd)

	BytomcliCmd.AddCommand(signMsgCmd)
	BytomcliCmd.AddCommand(verifyMsgCmd)
	BytomcliCmd.AddCommand(decodeProgCmd)

	BytomcliCmd.AddCommand(createTransactionFeedCmd)
	BytomcliCmd.AddCommand(listTransactionFeedsCmd)
	BytomcliCmd.AddCommand(deleteTransactionFeedCmd)
	BytomcliCmd.AddCommand(getTransactionFeedCmd)
	BytomcliCmd.AddCommand(updateTransactionFeedCmd)

	BytomcliCmd.AddCommand(isMiningCmd)
	BytomcliCmd.AddCommand(setMiningCmd)

	BytomcliCmd.AddCommand(netInfoCmd)
	BytomcliCmd.AddCommand(gasRateCmd)

	BytomcliCmd.AddCommand(versionCmd)
}

// AddTemplateFunc adds usage template to the root command BytomcliCmd.
func AddTemplateFunc() {
	walletEnableCmd := []string{
		createAccountCmd.Name(),
		listAccountsCmd.Name(),
		deleteAccountCmd.Name(),
		updateAccountAliasCmd.Name(),
		createAccountReceiverCmd.Name(),
		listAddressesCmd.Name(),
		validateAddressCmd.Name(),
		listPubKeysCmd.Name(),

		createAssetCmd.Name(),
		getAssetCmd.Name(),
		listAssetsCmd.Name(),
		updateAssetAliasCmd.Name(),

		createKeyCmd.Name(),
		deleteKeyCmd.Name(),
		listKeysCmd.Name(),
		resetKeyPwdCmd.Name(),
		checkKeyPwdCmd.Name(),
		signMsgCmd.Name(),

		buildTransactionCmd.Name(),
		signTransactionCmd.Name(),

		getTransactionCmd.Name(),
		listTransactionsCmd.Name(),
		listUnspentOutputsCmd.Name(),
		listBalancesCmd.Name(),

		rescanWalletCmd.Name(),
		walletInfoCmd.Name(),
	}

	cobra.AddTemplateFunc("WalletEnable", func(cmdName string) bool {
		for _, name := range walletEnableCmd {
			if name == cmdName {
				return true
			}
		}
		return false
	})

	cobra.AddTemplateFunc("WalletDisable", func(cmdName string) bool {
		for _, name := range walletEnableCmd {
			if name == cmdName {
				return false
			}
		}
		return true
	})
}
