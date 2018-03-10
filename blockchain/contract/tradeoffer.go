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
	TradeOfferEnding string = "1a000000"
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
func (a *TradeOffer) Build() (buildReqStr string, err error) {
	switch a.Selector {
	case ClauseTrade:
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
	case ClauseCancel:
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.AccountInfo, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.AccountInfo, a.BtmGas, a.AccountInfo)
		}
	default:
		err = errors.WithDetailf(ErrBadClause, "selected clause [%v] error, contract TradeOffer's clause must in set:[%v, %v]",
			a.Selector, ClauseTrade, ClauseCancel)
	}

	return
}

// AddArgs add the parameters for contract
func (a *TradeOffer) AddArgs(tpl *txbuilder.Template) (err error) {
	switch a.Selector {
	case ClauseTrade:
		err = addDataArgs(tpl, []string{a.Selector})
	case ClauseCancel:
		pubKeyInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
		paramInfo := NewParamInfo(nil, []PubKeyInfo{pubKeyInfo}, []string{a.Selector})
		err = addParamArgs(tpl, paramInfo)
	default:
		err = errors.WithDetailf(ErrBadClause, "the selector[%s] for contract TradeOffer is wrong!", a.Selector)
	}

	return
}
