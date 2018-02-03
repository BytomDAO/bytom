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
	a := new(Escrow)
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
func (a *CallOption) Build() (*string, error) {
	var buildReqStr string
	var buf string

	if a.Selector == ClauseExercise {
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildInlineProgReqFmtByAlias, a.OutputID,
				a.InnerAssetInfo, a.InnerAmount, a.InnerProgram,
				a.AssetInfo, a.Amount, a.ControlProgram,
				a.InnerAssetInfo, a.InnerAmount, a.InnerAccountInfo,
				a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildInlineProgReqFmt, a.OutputID,
				a.InnerAssetInfo, a.InnerAmount, a.InnerProgram,
				a.AssetInfo, a.Amount, a.ControlProgram,
				a.InnerAssetInfo, a.InnerAmount, a.InnerAccountInfo,
				a.BtmGas, a.AccountInfo)
		}
	} else if a.Selector == ClauseExpire {
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		}
	} else {
		if a.Selector == CallOptionEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", a.Selector, ClauseExercise, ClauseExpire, CallOptionEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

// AddArgs add the parameters for contract
func (a *CallOption) AddArgs(tpl *txbuilder.Template) (*txbuilder.Template, error) {
	var err error

	if a.Selector == ClauseExercise {
		pubInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
		paramInfo := NewParamInfo(nil, []PubKeyInfo{pubInfo}, []string{a.Selector})

		if tpl, err = addParamArgs(tpl, paramInfo); err != nil {
			return nil, err
		}
	} else if a.Selector == ClauseExpire {
		if tpl, err = addDataArgs(tpl, []string{a.Selector}); err != nil {
			return nil, err
		}
	} else {
		buf := fmt.Sprintf("the arguments of contract 'CallOption' is not right, Please follow the prompts to add parameters!")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
