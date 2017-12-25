package commands

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/txbuilder"
)

func init() {
	buildTransaction.PersistentFlags().StringVarP(&buildType, "type", "t", "",
		"transaction type, valid types: 'issue', 'spend'")
	buildTransaction.PersistentFlags().StringVarP(&receiverProgram, "receiver", "r",
		"", "program of receiver")
	buildTransaction.PersistentFlags().StringVarP(&btmGas, "gas", "g",
		"20000000", "program of receiver")
	buildTransaction.PersistentFlags().BoolVar(&pretty, "pretty", false,
		"pretty print json result")
	SignTransactionCmd.PersistentFlags().StringVarP(&password, "password", "p", "",
		"password of the account which sign these transaction(s)")
	SignTransactionCmd.PersistentFlags().BoolVar(&pretty, "pretty", false,
		"pretty print json result")
}

var (
	buildType       string
	btmGas          string
	receiverProgram string
	password        string
	pretty          bool
)

var buildIssueReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "issue", "asset_id": "%s", "amount": %s},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"}
	]}`

var buildSpendReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_receiver", "asset_id": "%s", "amount": %s, "receiver":{"control_program": "%s","expires_at":"2017-12-28T12:52:06.78309768+08:00"}}
	]}`

var buildTransaction = &cobra.Command{
	Use:   "build-transaction <accountID> <assetID> <amount>",
	Short: "Build one transaction template",
	Args:  cobra.RangeArgs(3, 4),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("type")
		if buildType == "spend" {
			cmd.MarkFlagRequired("receiver")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var buildReqStr string
		accountID := args[0]
		assetID := args[1]
		amount := args[2]
		switch buildType {
		case "issue":
			buildReqStr = fmt.Sprintf(buildIssueReqFmt, btmGas, accountID, assetID, amount, assetID, amount, accountID)
		case "spend":
			buildReqStr = fmt.Sprintf(buildSpendReqFmt, btmGas, accountID, assetID, amount, accountID, assetID, amount, receiverProgram)
		default:
			jww.ERROR.Println("Invalid transaction template type")
			os.Exit(ErrLocalExe)
		}

		var buildReq blockchain.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &buildReq); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalExe)
		}

		data, exitCode := clientCall("/build-transaction", &buildReq)
		if exitCode != Success {
			os.Exit(exitCode)
		}
		rawTemplate, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalParse)
		}

		if pretty {
			var prettyJSON bytes.Buffer
			err := json.Indent(&prettyJSON, rawTemplate, "", " ")
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(ErrLocalParse)
			}
			jww.FEEDBACK.Printf("Template Type: %s\n%s\n", buildType, prettyJSON.String())
			return
		}
		jww.FEEDBACK.Printf("Template Type: %s\n%s\n", buildType, rawTemplate)
	},
}

var SignTransactionCmd = &cobra.Command{
	Use:   "sign-transaction  <json templates>",
	Short: "Sign transaction templates with account password",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("password")
	},
	Run: func(cmd *cobra.Command, args []string) {
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalExe)
		}

		var req = struct {
			Auth string
			Txs  txbuilder.Template `json:"transaction"`
		}{Auth: password, Txs: template}

		jww.FEEDBACK.Printf("\n\n")
		data, exitCode := clientCall("/sign-transaction", &req)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawSign, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalParse)
		}

		if pretty {
			var prettyJSON bytes.Buffer
			err := json.Indent(&prettyJSON, rawSign, "", " ")
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(ErrLocalParse)
			}
			jww.FEEDBACK.Printf("\nSign Template:\n%s\n", prettyJSON.String())
			return
		}

		jww.FEEDBACK.Printf("\nSign Template:\n%s\n", string(rawSign))
	},
}

var SubmitTransactionCmd = &cobra.Command{
	Use:   "submit-transaction  <signed json template>",
	Short: "Submit signed transaction template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalExe)
		}

		jww.FEEDBACK.Printf("\n\n")
		data, exitCode := clientCall("/submit-transaction", &template)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		result := data.(map[string]interface{})
		jww.FEEDBACK.Printf("\nSubmit txid:%v\n", result["txid"])
	},
}

var listTransactions = &cobra.Command{
	Use:   "list-transactions",
	Short: "List the transactions",
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
		data, exitCode := clientCall("/list-transactions", &in)
		if exitCode != Success {
			os.Exit(exitCode)
		}

		rawPage, err := base64.StdEncoding.DecodeString(data.(string))
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalParse)
		}

		if err := json.Unmarshal(rawPage, &response); err != nil {
			jww.ERROR.Println(err)
			os.Exit(ErrLocalParse)
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

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/gas-rate")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
