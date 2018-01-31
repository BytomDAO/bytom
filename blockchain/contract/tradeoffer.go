package contract

import (
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseTrade is the contract TradeOffer's clause trade
	ClauseTrade string = "00000000"
	//ClauseCancel is the contract TradeOffer's clause cancel
	ClauseCancel string = "13000000"
)

func buildTradeOfferReq(args []string, minArgsCount int, alias bool, btmGas string) (*string, error) {
	var buildReqStr string
	var buf string

	outputID := args[0]
	accountInfo := args[1]
	assetInfo := args[2]
	amount := args[3]
	selector := args[minArgsCount]
	ClauseEnding := "1a000000"

	if selector == ClauseTrade {
		if len(args) != minArgsCount+5 {
			buf = fmt.Sprintf("the number of arguments[%d] for clause 'trade' in contract 'TradeOffer' is not equal to 5", len(args)-minArgsCount)
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
	} else if selector == ClauseCancel {
		if len(args) != minArgsCount+4 {
			buf := fmt.Sprintf("the number of arguments[%d] for clause 'cancel' in contract 'TradeOffer' is not equal to 4", len(args)-minArgsCount)
			err := errors.New(buf)
			return nil, err
		}

		if alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		}
	} else {
		if selector == ClauseEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseTrade, ClauseCancel, ClauseEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

func addTradeOfferArgs(tpl *txbuilder.Template, contractArgs []string) (*txbuilder.Template, error) {
	var err error

	if len(contractArgs) == 5 && contractArgs[0] == ClauseTrade {
		if tpl, err = addDataArgs(tpl, []string{contractArgs[0]}); err != nil {
			return nil, err
		}
	} else if len(contractArgs) == 4 && contractArgs[0] == ClauseCancel {
		pubInfo := newPubKeyInfo(contractArgs[1], []string{contractArgs[2], contractArgs[3]})
		paramInfo := newParamInfo(nil, []PubKeyInfo{pubInfo}, []string{contractArgs[0]})

		if tpl, err = addParamArgs(tpl, paramInfo); err != nil {
			return nil, err
		}
	} else {
		buf := fmt.Sprintf("the arguments of contract 'TradeOffer' is not right, Please follow the prompts to add parameters!")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
