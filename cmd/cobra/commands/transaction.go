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
)

//BTMGAS
const BTMGAS = "2000000"

func init() {
	buildTransaction.PersistentFlags().StringVarP(&buildType, "type", "t", "",
		"build transaction template type,invalid 'issue','spend','receiver'")
	buildTransaction.PersistentFlags().StringVarP(&receiverAccountID, "account", "a",
		"", "accountID of receiver")
	buildTransaction.PersistentFlags().StringVarP(&receiverProgram, "receiver", "r",
		"", "program of receiver")
	buildTransaction.PersistentFlags().BoolVarP(&pretty, "pretty", "p", false,
		"pretty print json result")
}

var (
	buildType         string
	receiverAccountID string
	receiverProgram   string
	pretty            bool
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
		{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"}
	]}`

var buildReceiverReqFmt = `
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
		switch buildType {
		case "spend":
			cmd.MarkFlagRequired("account")
		case "receiver":
			cmd.MarkFlagRequired("program")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var buildReqStr string
		accountID := args[0]
		assetID := args[1]
		amount := args[2]
		switch buildType {
		case "issue":
			buildReqStr = fmt.Sprintf(buildIssueReqFmt, BTMGAS, accountID, assetID, amount, assetID, amount, accountID)
		case "spend":
			buildReqStr = fmt.Sprintf(buildSpendReqFmt, BTMGAS, accountID, assetID, amount, accountID, assetID, amount, receiverAccountID)
		case "receiver":
			buildReqStr = fmt.Sprintf(buildReceiverReqFmt, BTMGAS, accountID, assetID, amount, accountID, assetID, amount, receiverProgram)
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
			os.Exit(ErrLocalUnwrap)
		}

		if pretty {
			var prettyJSON bytes.Buffer
			err := json.Indent(&prettyJSON, rawTemplate, "", " ")
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(ErrLocalUnwrap)
			}
			jww.FEEDBACK.Printf("Template Type: %s\n%s\n", buildType, prettyJSON.String())
			return
		}
		jww.FEEDBACK.Printf("Template Type: %s\n%s\n", buildType, string(rawTemplate))
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

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := clientCall("/gas-rate")
		if exitCode != Success {
			os.Exit(exitCode)
		}
		jww.FEEDBACK.Printf("gas rate: %v\n", data)
	},
}
