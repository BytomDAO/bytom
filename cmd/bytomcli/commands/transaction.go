package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/contract"
	"github.com/bytom/util"
)

func init() {
	buildTransactionCmd.PersistentFlags().StringVarP(&buildType, "type", "t", "", "transaction type, valid types: 'issue', 'spend', 'contract'")
	buildTransactionCmd.PersistentFlags().StringVarP(&receiverProgram, "receiver", "r", "", "program of receiver")
	buildTransactionCmd.PersistentFlags().StringVarP(&contractProgram, "contract", "c", "", "program of contract")
	buildTransactionCmd.PersistentFlags().StringVarP(&address, "address", "a", "", "address of receiver")
	buildTransactionCmd.PersistentFlags().StringVarP(&btmGas, "gas", "g", "20000000", "program of receiver")
	buildTransactionCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "pretty print json result")
	buildTransactionCmd.PersistentFlags().BoolVar(&alias, "alias", false, "use alias build transaction")

	buildContractTransactionCmd.PersistentFlags().StringVarP(&btmGas, "gas", "g", "20000000", "program of receiver")
	buildContractTransactionCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "pretty print json result")
	buildContractTransactionCmd.PersistentFlags().BoolVar(&alias, "alias", false, "use alias build transaction")
	buildContractTransactionCmd.PersistentFlags().StringVarP(&contractName, "contract-name", "c", "",
		"name of template contract, currently supported: 'LockWithPublicKey', 'LockWithMultiSig', 'LockWithPublicKeyHash',"+
			"\n\t\t\t       'RevealPreimage', 'TradeOffer', 'Escrow', 'CallOption', 'LoanCollateral'")

	signTransactionCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "password of the account which sign these transaction(s)")
	signTransactionCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "pretty print json result")

	signSubTransactionCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "password of the account which sign these transaction(s)")

	listTransactionsCmd.PersistentFlags().StringVar(&txID, "id", "", "transaction id")
	listTransactionsCmd.PersistentFlags().StringVar(&account, "account_id", "", "account id")
	listTransactionsCmd.PersistentFlags().BoolVar(&detail, "detail", false, "list transactions details")
}

var (
	buildType       = ""
	btmGas          = ""
	receiverProgram = ""
	contractProgram = ""
	address         = ""
	password        = ""
	pretty          = false
	alias           = false
	txID            = ""
	account         = ""
	detail          = false
	contractName    = ""
)

var buildIssueReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "issue", "asset_id": "%s", "amount": %s},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"}
	]}`

var buildIssueReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account", "asset_alias": "btm", "amount":%s, "account_alias": "%s"},
		{"type": "issue", "asset_alias": "%s", "amount": %s},
		{"type": "control_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"}
	]}`

var buildSpendReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_receiver", "asset_id": "%s", "amount": %s, "receiver":{"control_program": "%s","expires_at":"2017-12-28T12:52:06.78309768+08:00"}}
	]}`

var buildSpendReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account", "asset_alias": "btm", "amount":%s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "%s","amount": %s,"account_alias": "%s"},
		{"type": "control_receiver", "asset_alias": "%s", "amount": %s, "receiver":{"control_program": "%s","expires_at":"2017-12-28T12:52:06.78309768+08:00"}}
	]}`

var buildRetireReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "retire", "asset_id": "%s","amount": %s,"account_id": "%s"}
	]}`

var buildRetireReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account", "asset_alias": "btm", "amount":%s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "%s","amount": %s,"account_alias": "%s"},
		{"type": "retire", "asset_alias": "%s","amount": %s,"account_alias": "%s"}
	]}`

var buildControlAddressReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_address", "asset_id": "%s", "amount": %s,"address": "%s"}
	]}`

var buildControlAddressReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account", "asset_alias": "btm", "amount":%s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "%s","amount": %s, "account_alias": "%s"},
		{"type": "control_address", "asset_alias": "%s", "amount": %s,"address": "%s"}
	]}`

var buildContractReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s", "reference_data": {}}
	]}`

var buildContractReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account", "asset_alias": "btm", "amount":%s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "%s","amount": %s,"account_alias": "%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s", "reference_data": {}}
	]}`

var buildTransactionCmd = &cobra.Command{
	Use:   "build-transaction <accountID|alias> <assetID|alias> <amount>",
	Short: "Build one transaction template,default use account id and asset id",
	Args:  cobra.RangeArgs(3, 4),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("type")
		if buildType == "spend" {
			cmd.MarkFlagRequired("receiver")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var buildReqStr string
		accountInfo := args[0]
		assetInfo := args[1]
		amount := args[2]
		switch buildType {
		case "issue":
			if alias {
				buildReqStr = fmt.Sprintf(buildIssueReqFmtByAlias, btmGas, accountInfo, assetInfo, amount, assetInfo, amount, accountInfo)
				break
			}
			buildReqStr = fmt.Sprintf(buildIssueReqFmt, btmGas, accountInfo, assetInfo, amount, assetInfo, amount, accountInfo)
		case "spend":
			if alias {
				buildReqStr = fmt.Sprintf(buildSpendReqFmtByAlias, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, receiverProgram)
				break
			}
			buildReqStr = fmt.Sprintf(buildSpendReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, receiverProgram)
		case "retire":
			if alias {
				buildReqStr = fmt.Sprintf(buildRetireReqFmtByAlias, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, accountInfo)
				break
			}
			buildReqStr = fmt.Sprintf(buildRetireReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, accountInfo)
		case "address":
			if alias {
				buildReqStr = fmt.Sprintf(buildControlAddressReqFmtByAlias, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, address)
				break
			}
			buildReqStr = fmt.Sprintf(buildControlAddressReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, address)
		case "contract":
			if alias {
				buildReqStr = fmt.Sprintf(buildContractReqFmtByAlias, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, contractProgram)
				break
			}
			buildReqStr = fmt.Sprintf(buildContractReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, contractProgram)
		default:
			jww.ERROR.Println("Invalid transaction template type")
			os.Exit(util.ErrLocalExe)
		}

		var buildReq blockchain.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &buildReq); err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		data, exitCode := util.ClientCall("/build-transaction", &buildReq)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		if pretty {
			printJSON(data)
			return
		}

		dataMap, ok := data.(map[string]interface{})
		if ok != true {
			jww.ERROR.Println("invalid type assertion")
			os.Exit(util.ErrLocalParse)
		}

		rawTemplate, err := json.Marshal(dataMap)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}

		jww.FEEDBACK.Printf("Template Type: %s\n%s\n", buildType, string(rawTemplate))
	},
}

var buildContractTransactionCmd = &cobra.Command{
	Use:   "build-contract-transaction <outputID> <accountID|alias> <assetID|alias> <amount> -c <contractName> <contractArgs>",
	Short: "Build transaction for template contract, default use account id and asset id",
	Args:  cobra.RangeArgs(4, 20),
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.MarkFlagRequired("contract-name")
	},
	Run: func(cmd *cobra.Command, args []string) {
		var buildReqStr string
		var err error

		minArgsCount := 4
		usage := "Usage:\n  bytomcli build-contract-transaction <outputID> <accountID|alias> <assetID|alias> <amount> -c <contractName>"

		if ok := contract.CheckContractArgs(contractName, args, minArgsCount, usage); !ok {
			os.Exit(util.ErrLocalExe)
		}

		buildReqStr, err = contract.BuildContractTransaction(contractName, args, minArgsCount, alias, btmGas)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}
		fmt.Println("buildReqStr", buildReqStr)

		var buildReq blockchain.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &buildReq); err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		data, exitCode := util.ClientCall("/build-transaction", &buildReq)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		dataMap, ok := data.(map[string]interface{})
		if ok != true {
			jww.ERROR.Println("invalid type assertion")
			os.Exit(util.ErrLocalParse)
		}

		rawTemplate, err := json.Marshal(dataMap)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}

		/********************add arguments for contract*********************/
		var tpl *txbuilder.Template
		err = json.Unmarshal(rawTemplate, &tpl)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}

		var contractArgs []string
		count := minArgsCount
		for count < len(args) {
			contractArgs = append(contractArgs, args[count])
			count++
		}

		tpl, err = contract.AddContractArguments(tpl, contractName, contractArgs)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}

		addWitnessTemplate, err := json.Marshal(tpl)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}

		jww.FEEDBACK.Printf("\ntxbuilder.Template: \n%s\n", string(addWitnessTemplate))
	},
}

var signTransactionCmd = &cobra.Command{
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
			os.Exit(util.ErrLocalExe)
		}

		var req = struct {
			Auth string
			Txs  txbuilder.Template `json:"transaction"`
		}{Auth: "123456", Txs: template}

		jww.FEEDBACK.Printf("\n\n")
		data, exitCode := util.ClientCall("/sign-transaction", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		if pretty {
			printJSON(data)
			return
		}

		dataMap, ok := data.(map[string]interface{})
		if ok != true {
			jww.ERROR.Println("invalid type assertion")
			os.Exit(util.ErrLocalParse)
		}

		rawSign, err := json.Marshal(dataMap)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}
		jww.FEEDBACK.Printf("\nSign Template:\n%s\n", string(rawSign))
	},
}

var submitTransactionCmd = &cobra.Command{
	Use:   "submit-transaction  <signed json template>",
	Short: "Submit signed transaction template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		jww.FEEDBACK.Printf("\n\n")
		data, exitCode := util.ClientCall("/submit-transaction", &template)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var signSubTransactionCmd = &cobra.Command{
	Use:   "sign-submit-transaction  <json templates>",
	Short: "Sign and Submit transaction templates with account password",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// cmd.MarkFlagRequired("password")
	},
	Run: func(cmd *cobra.Command, args []string) {
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		var req = struct {
			Auth string
			Txs  txbuilder.Template `json:"transaction"`
		}{Auth: "123456", Txs: template}

		jww.FEEDBACK.Printf("\n\n")
		data, exitCode := util.ClientCall("/sign-submit-transaction", &req)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSON(data)
	},
}

var listTransactionsCmd = &cobra.Command{
	Use:   "list-transactions",
	Short: "List the transactions",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		filter := struct {
			ID        string `json:"id"`
			AccountID string `json:"account_id"`
			Detail    bool   `json:"detail"`
		}{ID: txID, AccountID: account, Detail: detail}

		data, exitCode := util.ClientCall("/list-transactions", &filter)
		if exitCode != util.Success {
			os.Exit(exitCode)
		}

		printJSONList(data)
	},
}

var gasRateCmd = &cobra.Command{
	Use:   "gas-rate",
	Short: "Print the current gas rate",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, exitCode := util.ClientCall("/gas-rate")
		if exitCode != util.Success {
			os.Exit(exitCode)
		}
		printJSON(data)
	},
}
