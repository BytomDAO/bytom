package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/util"
	"github.com/spf13/cobra"
	"github.com/bytom/contract"
	jww "github.com/spf13/jwalterweatherman"
)

func init() {
	buildContractTransactionCmd.PersistentFlags().StringVarP(&btmGas, "gas", "g", "20000000", "program of receiver")
	buildContractTransactionCmd.PersistentFlags().BoolVar(&alias, "alias", false, "use alias build transaction")
}

var buildContractTransactionCmd = &cobra.Command{
	Use:   "build-contract-transaction <contractName> <outputID> <accountID|alias> <assetID|alias> <amount> <contractArgs>",
	Short: "Build transaction for template contract, default use account id and asset id",
	Args:  cobra.RangeArgs(1, 20),
	Run: func(cmd *cobra.Command, args []string) {
		var buildReqStr string
		var err error

		contractName := args[0]
		minArgsCount := 5
		Usage := "Usage:\n  bytomcli build-contract-transaction <contractName> <outputID> <accountID|alias> <assetID|alias> <amount>"

		if ok := contract.CheckContractArgs(contractName, args, minArgsCount, Usage); !ok {
			os.Exit(util.ErrLocalExe)
		}

		buildReqStr, err = contract.BuildContractTransaction(args, minArgsCount, alias, btmGas)
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
