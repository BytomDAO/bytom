package contract

import (
	"encoding/json"
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseTrade is the contract TradeOffer's clause trade
	ClauseTrade string = "00000000"
	//ClauseCancel is the contract TradeOffer's clause cancel
	ClauseCancel string = "13000000"
	//TradeOfferEnding is the contract TradeOffer's clause ending
	TradeOfferEnding string = "13000000"
)

// TradeOffer stores the information of TradeOffer contract
type TradeOffer struct {
	CommonInfo
	Selector string `json:"selector"`
	PaymentInfo
	PubKeyInfo
}

// DecodeTradeOffer unmarshal JSON-encoded data of contract action
func DecodeTradeOffer(data []byte) (ContractAction, error) {
	a := new(TradeOffer)
	err := json.Unmarshal(data, a)
	return a, err
}

// BuildContractReq create new ContractReq which contain contract's name and arguments
func (a *TradeOffer) BuildContractReq(contractName string) (*ContractReq, error) {
	arguments, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	return &ContractReq{
		ContractName: contractName,
		ContractArgs: arguments,
	}, nil
}

// Build create a transaction request
func (a *TradeOffer) Build() (*string, error) {
	var buildReqStr string
	var buf string

	if a.Selector == ClauseTrade {
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildInlineAcctReqFmtByAlias, a.OutputID,
				a.InnerAssetInfo, a.InnerAmount, a.InnerProgram,
				a.InnerAssetInfo, a.InnerAmount, a.InnerAccountInfo,
				a.BtmGas, a.AccountInfo,
				a.AssetInfo, a.Amount, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildInlineAcctReqFmt, a.OutputID,
				a.InnerAssetInfo, a.InnerAmount, a.InnerProgram,
				a.InnerAssetInfo, a.InnerAmount, a.InnerAccountInfo,
				a.BtmGas, a.AccountInfo,
				a.AssetInfo, a.Amount, a.AccountInfo)
		}
	} else if a.Selector == ClauseCancel {
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.AccountInfo, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.AccountInfo, a.BtmGas, a.AccountInfo)
		}
	} else {
		if a.Selector == TradeOfferEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", a.Selector, ClauseTrade, ClauseCancel, TradeOfferEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

// AddArgs add the parameters for contract
func (a *TradeOffer) AddArgs(tpl *txbuilder.Template) (*txbuilder.Template, error) {
	var err error

	if a.Selector == ClauseTrade {
		if tpl, err = addDataArgs(tpl, []string{a.Selector}); err != nil {
			return nil, err
		}
	} else if a.Selector == ClauseCancel {
		pubKeyInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
		paramInfo := NewParamInfo(nil, []PubKeyInfo{pubKeyInfo}, []string{a.Selector})

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
