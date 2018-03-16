package commands

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/bytom/util"
)

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
			cmd.Usage()
		}
	},
}

// Execute adds all child commands to the root command BytomcliCmd and sets flags appropriately.
func Execute() {

	AddCommands()

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
	BytomcliCmd.AddCommand(updateAccountTagsCmd)
	BytomcliCmd.AddCommand(createAccountReceiverCmd)

	BytomcliCmd.AddCommand(createAssetCmd)
	BytomcliCmd.AddCommand(listAssetsCmd)
	BytomcliCmd.AddCommand(updateAssetTagsCmd)
	BytomcliCmd.AddCommand(updateAssetAliasCmd)

	BytomcliCmd.AddCommand(listTransactionsCmd)
	BytomcliCmd.AddCommand(listUnspentOutputsCmd)
	BytomcliCmd.AddCommand(listBalancesCmd)

	BytomcliCmd.AddCommand(buildTransactionCmd)
	BytomcliCmd.AddCommand(signTransactionCmd)
	BytomcliCmd.AddCommand(submitTransactionCmd)
	BytomcliCmd.AddCommand(signSubTransactionCmd)

	BytomcliCmd.AddCommand(blockHeightCmd)
	BytomcliCmd.AddCommand(blockHashCmd)
	BytomcliCmd.AddCommand(getBlockByHashCmd)
	BytomcliCmd.AddCommand(getBlockHeaderByHashCmd)
	BytomcliCmd.AddCommand(getBlockTransactionsCountByHashCmd)
	BytomcliCmd.AddCommand(getBlockByHeightCmd)
	BytomcliCmd.AddCommand(getBlockTransactionsCountByHeightCmd)

	BytomcliCmd.AddCommand(createKeyCmd)
	BytomcliCmd.AddCommand(deleteKeyCmd)
	BytomcliCmd.AddCommand(listKeysCmd)
	BytomcliCmd.AddCommand(exportPrivateCmd)
	BytomcliCmd.AddCommand(importPrivateCmd)
	BytomcliCmd.AddCommand(importKeyProgressCmd)

	BytomcliCmd.AddCommand(isMiningCmd)

	BytomcliCmd.AddCommand(netInfoCmd)
	BytomcliCmd.AddCommand(netListeningCmd)
	BytomcliCmd.AddCommand(peerCountCmd)
	BytomcliCmd.AddCommand(netSyncingCmd)

	BytomcliCmd.AddCommand(gasRateCmd)

	BytomcliCmd.AddCommand(createTransactionFeedCmd)
	BytomcliCmd.AddCommand(listTransactionFeedsCmd)
	BytomcliCmd.AddCommand(deleteTransactionFeedCmd)
	BytomcliCmd.AddCommand(getTransactionFeedCmd)
	BytomcliCmd.AddCommand(updateTransactionFeedCmd)

	BytomcliCmd.AddCommand(versionCmd)
}
