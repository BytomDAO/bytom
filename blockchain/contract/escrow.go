package contract

import (
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseApprove is the contract Escrow's clause approve
	ClauseApprove string = "00000000"
	//ClauseReject is the contract Escrow's clause reject
	ClauseReject string = "1b000000"
)

func buildEscrowReq(args []string, minArgsCount int, alias bool, btmGas string) (*string, error) {
	var buildReqStr string
	var buf string

	outputID := args[0]
	accountInfo := args[1]
	assetInfo := args[2]
	amount := args[3]
	selector := args[minArgsCount]
	ClauseEnding := "2a000000"

	if selector == ClauseApprove || selector == ClauseReject {
		if len(args) != minArgsCount+5 {
			buf = fmt.Sprintf("the number of arguments[%d] for clause 'approve' or 'reject' in contract 'Escrow' is not equal to 5", len(args)-minArgsCount)
			err := errors.New(buf)
			return nil, err
		}

		controlProgram := args[minArgsCount+4]
		if alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
		}
	} else {
		if selector == ClauseEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseApprove, ClauseReject, ClauseEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

func addEscrowArgs(tpl *txbuilder.Template, contractArgs []string) (*txbuilder.Template, error) {
	var err error

	if len(contractArgs) == 5 && (contractArgs[0] == ClauseApprove || contractArgs[0] == ClauseReject) {
		pubInfo := newPubKeyInfo(contractArgs[1], []string{contractArgs[2], contractArgs[3]})
		paramInfo := newParamInfo(nil, []PubKeyInfo{pubInfo}, []string{contractArgs[0]})

		if tpl, err = addParamArgs(tpl, paramInfo); err != nil {
			return nil, err
		}
	} else {
		buf := fmt.Sprintf("the arguments of contract 'Escrow' is not right, Please follow the prompts to add parameters!")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
