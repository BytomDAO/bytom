package commands

import (
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
		jww.FEEDBACK.Println(data[0])
	},
}

var listAccessTokenCmd = &cobra.Command{
	Use:   "list-access-tokens",
	Short: "List the existing access tokens",
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
		data, exitCode := clientCall("/list-access-tokens", &in)
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

var deleteAccessTokenCmd = &cobra.Command{
	Use:   "delete-access-token",
	Short: "delete an access token",
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
	Short: "check an access token",
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
