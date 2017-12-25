package commands

import (
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var createAccessTokenCmd = &cobra.Command{
	Use:   "create-access-token",
	Short: "Create a new access token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var token accessToken
		token.ID = args[0]

		data, exitCode := clientCall("/create-access-token", &token)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Println(data)
	},
}

var listAccessTokenCmd = &cobra.Command{
	Use:   "list-access-tokens",
	Short: "List the existing access tokens",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		data, exitCode := clientCall("/list-access-tokens")
		if exitCode != Success {
			os.Exit(exitCode)
		}

		tokenList := data.([]interface{})
		for idx, item := range tokenList {
			token, err := json.MarshalIndent(item, "", " ")
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(ErrLocalParse)
			}
			jww.FEEDBACK.Printf("%d:\n%v\n\n", idx, token)
		}
	},
}

var deleteAccessTokenCmd = &cobra.Command{
	Use:   "delete-access-token",
	Short: "Delete an access token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var token accessToken
		token.ID = args[0]

		_, exitCode := clientCall("/delete-access-token", &token)
		if exitCode != Success {
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

		_, exitCode := clientCall("/check-access-token", &token)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		jww.FEEDBACK.Println("Valid access token")
	},
}
