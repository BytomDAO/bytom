package commands

import (
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/bytom/util"
)

var createAccessTokenCmd = &cobra.Command{
	Use:   "create-access-token <tokenID>",
	Short: "Create a new access token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var token accessToken
		token.ID = args[0]

		data, exitCode := util.ClientCall("/create-access-token", &token)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}

var listAccessTokenCmd = &cobra.Command{
	Use:   "list-access-tokens",
	Short: "List the existing access tokens",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/list-access-tokens")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var deleteAccessTokenCmd = &cobra.Command{
	Use:   "delete-access-token <tokenID>",
	Short: "Delete an access token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var token accessToken
		token.ID = args[0]

		if _, exitCode := util.ClientCall("/delete-access-token", &token); exitCode != util.Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println("Successfully delete access token")
	},
}

var checkAccessTokenCmd = &cobra.Command{
	Use:   "check-access-token <tokenID> <secret>",
	Short: "Check an access token",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var token accessToken
		token.ID = args[0]
		token.Secret = args[1]

		if _, exitCode := util.ClientCall("/check-access-token", &token); exitCode != util.Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Valid access token")
	},
}
