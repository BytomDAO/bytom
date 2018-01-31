package contract

import (
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseExercise is the contract CallOption's clause exercise
	ClauseExercise string = "00000000"
	//ClauseExpire is the contract CallOption's clause expire
	ClauseExpire string = "22000000"
)

func buildCallOptionReq(args []string, minArgsCount int, alias bool, btmGas string) (*string, error) {
	var buildReqStr string
	var buf string

	outputID := args[0]
	accountInfo := args[1]
	assetInfo := args[2]
	amount := args[3]
	selector := args[minArgsCount]
	ClauseEnding := "2f000000"

	if selector == ClauseExercise {
		if len(args) != minArgsCount+8 {
			buf = fmt.Sprintf("the number of arguments[%d] for clause 'exercise' in contract 'CallOption' is not equal to 8", len(args)-minArgsCount)
			err := errors.New(buf)
			return nil, err
		}

		innerAccountInfo := args[minArgsCount+1]
		innerAssetInfo := args[minArgsCount+2]
		innerAmount := args[minArgsCount+3]
		innerProgram := args[minArgsCount+4]
		if alias {
			buildReqStr = fmt.Sprintf(buildInlineAcctReqFmtByAlias, outputID,
				innerAssetInfo, innerAmount, innerProgram,
				innerAssetInfo, innerAmount, innerAccountInfo,
				btmGas, accountInfo,
				assetInfo, amount, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildInlineAcctReqFmt, outputID,
				innerAssetInfo, innerAmount, innerProgram,
				innerAssetInfo, innerAmount, innerAccountInfo,
				btmGas, accountInfo,
				assetInfo, amount, accountInfo)
		}
	} else if selector == ClauseExpire {
		if len(args) != minArgsCount+2 {
			buf = fmt.Sprintf("the number of arguments[%d] for clause 'expire' in contract 'CallOption' is not equal to 2", len(args)-minArgsCount)
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
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseExercise, ClauseExpire, ClauseEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

func addCallOptionArgs(tpl *txbuilder.Template, contractArgs []string) (*txbuilder.Template, error) {
	var err error

	if len(contractArgs) == 8 && contractArgs[0] == ClauseExercise {
		pubInfo := newPubKeyInfo(contractArgs[5], []string{contractArgs[6], contractArgs[7]})
		paramInfo := newParamInfo(nil, []PubKeyInfo{pubInfo}, []string{contractArgs[0]})

		if tpl, err = addParamArgs(tpl, paramInfo); err != nil {
			return nil, err
		}
	} else if len(contractArgs) == 2 && contractArgs[0] == ClauseExpire {
		if tpl, err = addDataArgs(tpl, []string{contractArgs[0]}); err != nil {
			return nil, err
		}
	} else {
		buf := fmt.Sprintf("the arguments of contract 'CallOption' is not right, Please follow the prompts to add parameters!")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
