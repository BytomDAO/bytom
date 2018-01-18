package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/util"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

const (
	//ClauseTrade is the contract TradeOffer's clause trade
	ClauseTrade string = "00000000"
	//ClauseCancel is the contract TradeOffer's clause cancel
	ClauseCancel string = "13000000"
	//ClauseApprove is the contract Escrow's clause approve
	ClauseApprove string = "00000000"
	//ClauseReject is the contract Escrow's clause reject
	ClauseReject string = "1b000000"
	//ClauseRepay is the contract LoanCollateral's clause repay
	ClauseRepay string = "00000000"
	//ClauseDefault is the contract LoanCollateral's clause default
	ClauseDefault string = "1c000000"
)

func init() {
	buildContractTransactionCmd.PersistentFlags().StringVarP(&btmGas, "gas", "g", "20000000", "program of receiver")
	buildContractTransactionCmd.PersistentFlags().BoolVar(&alias, "alias", false, "use alias build transaction")
}

var buildAcctRecvReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildAcctRecvReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"}
	]}`

var buildProgRecvReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildProgRecvReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"}
	]}`

var buildTradeOfferReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"}
	]}`

var buildTradeOfferReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"},
		{"type": "control_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"}
	]}`

var buildLoanCollateralReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildLoanCollateralReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"}
	]}`

var buildContractTransactionCmd = &cobra.Command{
	Use:   "build-contract-transaction <contractName> <outputID> <accountID|alias> <assetID|alias> <amount> <contractArgs>",
	Short: "Build transaction for template contract, default use account id and asset id",
	Args:  cobra.RangeArgs(1, 20),
	Run: func(cmd *cobra.Command, args []string) {
		var buildReqStr string
		var rootPub string
		var path1 string
		var path2 string
		var rootPub1 string
		var path11 string
		var path12 string
		var pubKey string
		var value string
		var selector string
		var innerAssetInfo string
		var innerAmount string
		var innerAccountInfo string
		var program string
		var controlProgram string

		contractName := args[0]
		minArgsCount := 5
		Usage := "Usage:\n  bytomcli build-contract-transaction <contractName> <outputID> <accountID|alias> <assetID|alias> <amount>"
		if ok := checkContractArgs(contractName, args, minArgsCount, Usage); !ok {
			os.Exit(util.ErrLocalExe)
		}

		outputID := args[1]
		accountInfo := args[2]
		assetInfo := args[3]
		amount := args[4]

		fmt.Println("contractName", contractName)
		switch contractName {
		case "LockWithPublicKey":
			if alias {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			}

			rootPub = args[minArgsCount]
			path1 = args[minArgsCount+1]
			path2 = args[minArgsCount+2]

		case "LockWithMultiSig":
			if alias {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			}

			rootPub = args[minArgsCount]
			path1 = args[minArgsCount+1]
			path2 = args[minArgsCount+2]
			rootPub1 = args[minArgsCount+3]
			path11 = args[minArgsCount+4]
			path12 = args[minArgsCount+5]

		case "LockWithPublicKeyHash":
			if alias {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			}

			pubKey = args[minArgsCount]
			rootPub = args[minArgsCount+1]
			path1 = args[minArgsCount+2]
			path2 = args[minArgsCount+3]

		case "RevealPreimage":
			if alias {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			}

			value = args[minArgsCount]

		case "TradeOffer":
			ClauseEnding := "1a000000"
			selector = args[minArgsCount]

			if selector == ClauseTrade && len(args) == minArgsCount+5 {
				innerAssetInfo = args[minArgsCount+1]
				innerAmount = args[minArgsCount+2]
				innerAccountInfo = args[minArgsCount+3]
				program = args[minArgsCount+4]

				if alias {
					buildReqStr = fmt.Sprintf(buildTradeOfferReqFmtByAlias, outputID,
						innerAssetInfo, innerAmount, program,
						innerAssetInfo, innerAmount, innerAccountInfo,
						btmGas, accountInfo,
						assetInfo, amount, accountInfo)
				} else {
					buildReqStr = fmt.Sprintf(buildTradeOfferReqFmt, outputID,
						innerAssetInfo, innerAmount, program,
						innerAssetInfo, innerAmount, innerAccountInfo,
						btmGas, accountInfo,
						assetInfo, amount, accountInfo)
				}
			} else if selector == ClauseCancel && len(args) == minArgsCount+4 {
				rootPub = args[minArgsCount+1]
				path1 = args[minArgsCount+2]
				path2 = args[minArgsCount+3]
				if alias {
					buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
				} else {
					buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
				}
			} else if selector == ClauseEnding {
				fmt.Printf("no clause was selected in this program, ending exit!!!\n")
				os.Exit(0)
			} else {
				fmt.Printf("selected clause [%v] error, clause must in set:[%v, %v, %v]\n", selector, ClauseTrade, ClauseCancel, ClauseEnding)
				os.Exit(util.ErrLocalExe)
			}

		case "Escrow":
			ClauseEnding := "2a000000"
			selector = args[minArgsCount]

			if (selector == ClauseApprove || selector == ClauseReject) && len(args) == minArgsCount+5 {
				rootPub = args[minArgsCount+1]
				path1 = args[minArgsCount+2]
				path2 = args[minArgsCount+3]
				controlProgram = args[minArgsCount+4]
				if alias {
					buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
				} else {
					buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
				}
			} else if selector == ClauseEnding {
				fmt.Printf("no clause was selected in this program, ending exit!!!\n")
				os.Exit(0)
			} else {
				fmt.Printf("selected clause [%v] error, clause must in set:[%v, %v, %v]\n", selector, ClauseApprove, ClauseReject, ClauseEnding)
				os.Exit(util.ErrLocalExe)
			}

		case "LoanCollateral":
			ClauseEnding := "28000000"
			selector = args[minArgsCount]

			if selector == ClauseRepay && len(args) == minArgsCount+6 {
				innerAssetInfo = args[minArgsCount+1]
				innerAmount = args[minArgsCount+2]
				innerAccountInfo = args[minArgsCount+3]
				program = args[minArgsCount+4]
				controlProgram = args[minArgsCount+5]
				if alias {
					buildReqStr = fmt.Sprintf(buildLoanCollateralReqFmtByAlias, outputID,
						innerAssetInfo, innerAmount, program,
						assetInfo, amount, controlProgram,
						innerAssetInfo, innerAmount, innerAccountInfo,
						btmGas, accountInfo)
				} else {
					buildReqStr = fmt.Sprintf(buildLoanCollateralReqFmt, outputID,
						innerAssetInfo, innerAmount, program,
						assetInfo, amount, controlProgram,
						innerAssetInfo, innerAmount, innerAccountInfo,
						btmGas, accountInfo)
				}
			} else if selector == ClauseDefault && len(args) == minArgsCount+1 {
				fmt.Printf("the clause was not implemented \n")
				os.Exit(0)
			} else if selector == ClauseEnding {
				fmt.Printf("no clause was selected in this program, ending exit!!!\n")
				os.Exit(0)
			} else {
				fmt.Printf("selected clause [%v] error, clause must in set:[%v, %v, %v]\n", selector, ClauseRepay, ClauseDefault, ClauseEnding)
				os.Exit(util.ErrLocalExe)
			}

		case "CallOption":
			fmt.Printf("the clause was not implemented \n")
			os.Exit(0)

		default:
			jww.ERROR.Println("Invalid contract template name")
			os.Exit(util.ErrLocalExe)
		}

		fmt.Println("buildReqStr:", buildReqStr)
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

		//add params for Witness
		switch contractName {
		case "LockWithPublicKey":
			tpl, err = addPublicKeyWitness(tpl, rootPub, path1, path2)
		case "LockWithMultiSig":
			tpl, err = addMultiSigWitness(tpl, rootPub, path1, path2, rootPub1, path11, path12)
		case "LockWithPublicKeyHash":
			tpl, err = addPublicKeyHashWitness(tpl, pubKey, rootPub, path1, path2)
		case "RevealPreimage":
			tpl, err = addValueWitness(tpl, value)
		case "TradeOffer":
			if selector == ClauseTrade {
				tpl, err = addValueWitness(tpl, selector)
			} else if selector == ClauseCancel {
				tpl, err = addPubValueWitness(tpl, rootPub, path1, path2, selector)
			}
		case "Escrow":
			tpl, err = addPubValueWitness(tpl, rootPub, path1, path2, selector)
		case "LoanCollateral":
			if selector == ClauseRepay {
				tpl, err = addValueWitness(tpl, selector)
			} else if selector == ClauseDefault {
				// Not implemented
			}
		case "CallOption":
			// Not implemented
		default:
			jww.ERROR.Println("Invalid contract template name")
			os.Exit(util.ErrLocalExe)
		}

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
