package contract

import (
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseRepay is the contract LoanCollateral's clause repay
	ClauseRepay string = "00000000"
	//ClauseDefault is the contract LoanCollateral's clause default
	ClauseDefault string = "1c000000"
)

func buildLoanCollateralReq(args []string, minArgsCount int, alias bool, btmGas string) (*string, error) {
	var buildReqStr string
	var buf string

	outputID := args[0]
	accountInfo := args[1]
	assetInfo := args[2]
	amount := args[3]
	selector := args[minArgsCount]
	ClauseEnding := "28000000"

	if selector == ClauseRepay {
		if len(args) != minArgsCount+6 {
			buf = fmt.Sprintf("the number of arguments[%d] for clause 'repay' in contract 'LoanCollateral' is not equal to 6", len(args)-minArgsCount)
			err := errors.New(buf)
			return nil, err
		}

		innerAccountInfo := args[minArgsCount+1]
		innerAssetInfo := args[minArgsCount+2]
		innerAmount := args[minArgsCount+3]
		innerProgram := args[minArgsCount+4]
		controlProgram := args[minArgsCount+5]
		if alias {
			buildReqStr = fmt.Sprintf(buildInlineProgReqFmtByAlias, outputID,
				innerAssetInfo, innerAmount, innerProgram,
				assetInfo, amount, controlProgram,
				innerAssetInfo, innerAmount, innerAccountInfo,
				btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildInlineProgReqFmt, outputID,
				innerAssetInfo, innerAmount, innerProgram,
				assetInfo, amount, controlProgram,
				innerAssetInfo, innerAmount, innerAccountInfo,
				btmGas, accountInfo)
		}
	} else if selector == ClauseDefault {
		if len(args) != minArgsCount+2 {
			buf = fmt.Sprintf("the number of arguments[%d] for clause 'default' in contract 'LoanCollateral' is not equal to 2", len(args)-minArgsCount)
			err := errors.New(buf)
			return nil, err
		}

		controlProgram := args[minArgsCount+1]
		if alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
		}
	} else {
		if selector == ClauseEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseRepay, ClauseDefault, ClauseEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

func addLoanCollateralArgs(tpl *txbuilder.Template, contractArgs []string) (*txbuilder.Template, error) {
	var err error

	if len(contractArgs) == 6 && contractArgs[0] == ClauseRepay {
		if tpl, err = addDataArgs(tpl, []string{contractArgs[0]}); err != nil {
			return nil, err
		}
	} else if len(contractArgs) == 2 && contractArgs[0] == ClauseDefault {
		if tpl, err = addDataArgs(tpl, []string{contractArgs[0]}); err != nil {
			return nil, err
		}
	} else {
		buf := fmt.Sprintf("the arguments of contract 'LoanCollateral' is not right, Please follow the prompts to add parameters!")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
