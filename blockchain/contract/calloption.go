package contract

import (
	"encoding/json"
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseExercise is the contract CallOption's clause exercise
	ClauseExercise string = "00000000"
	//ClauseExpire is the contract CallOption's clause expire
	ClauseExpire string = "22000000"
	//CallOptionEnding is the contract CallOption's clause ending
	CallOptionEnding string = "2f000000"
)

// CallOption stores the information of CallOption contract.
type CallOption struct {
	CommonInfo
	Selector       string `json:"selector"`
	ControlProgram string `json:"control_program"`
	PaymentInfo
	PubKeyInfo
}

// DecodeCallOption unmarshal JSON-encoded data of contract action
func DecodeCallOption(data []byte) (ContractAction, error) {
	a := new(CallOption)
	err := json.Unmarshal(data, a)
	return a, err
}

// BuildContractReq create new ContractReq which contain contract's name and arguments
func (a *CallOption) BuildContractReq(contractName string) (*ContractReq, error) {
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
func (a *CallOption) Build() (buildReqStr string, err error) {
	switch a.Selector {
	case ClauseExercise:
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
	case ClauseExpire:
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		}
	default:
		err = errors.WithDetailf(ErrBadClause, "selected clause [%v] error, contract CallOption's clause must in set:[%v, %v]",
			a.Selector, ClauseExercise, ClauseExpire)
	}

	return
}

// AddArgs add the parameters for contract
func (a *CallOption) AddArgs(tpl *txbuilder.Template) (err error) {
	switch a.Selector {
	case ClauseExercise:
		pubInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
		paramInfo := NewParamInfo(nil, []PubKeyInfo{pubInfo}, []string{a.Selector})
		err = addParamArgs(tpl, paramInfo)
	case ClauseExpire:
		err = addDataArgs(tpl, []string{a.Selector})
	default:
		err = errors.WithDetailf(ErrBadClause, "the selector[%s] for contract CallOption is wrong!", a.Selector)
	}

	return
}
