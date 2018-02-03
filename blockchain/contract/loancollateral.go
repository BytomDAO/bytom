package contract

import (
	"encoding/json"
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseRepay is the contract LoanCollateral's clause repay
	ClauseRepay string = "00000000"
	//ClauseDefault is the contract LoanCollateral's clause default
	ClauseDefault string = "1c000000"
	//LoanCollateralEnding is the contract LoanCollateral's clause ending
	LoanCollateralEnding string = "28000000"
)

// LoanCollateral stores the information of LoanCollateral contract
type LoanCollateral struct {
	CommonInfo
	Selector       string `json:"selector"`
	ControlProgram string `json:"control_program"`
	PaymentInfo
}

// DecodeLoanCollateral unmarshal JSON-encoded data of contract action
func DecodeLoanCollateral(data []byte) (ContractAction, error) {
	a := new(LoanCollateral)
	err := json.Unmarshal(data, a)
	return a, err
}

// BuildContractReq create new ContractReq which contain contract's name and arguments
func (a *LoanCollateral) BuildContractReq(contractName string) (*ContractReq, error) {
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
func (a *LoanCollateral) Build() (*string, error) {
	var buildReqStr string
	var buf string

	if a.Selector == ClauseRepay {
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
	} else if a.Selector == ClauseDefault {
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		}
	} else {
		if a.Selector == LoanCollateralEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", a.Selector, ClauseRepay, ClauseDefault, LoanCollateralEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

// AddArgs add the parameters for contract
func (a *LoanCollateral) AddArgs(tpl *txbuilder.Template) (*txbuilder.Template, error) {
	var err error

	if a.Selector == ClauseRepay {
		if tpl, err = addDataArgs(tpl, []string{a.Selector}); err != nil {
			return nil, err
		}
	} else if a.Selector == ClauseDefault {
		if tpl, err = addDataArgs(tpl, []string{a.Selector}); err != nil {
			return nil, err
		}
	} else {
		buf := fmt.Sprintf("the arguments of contract 'LoanCollateral' is not right, Please follow the prompts to add parameters!")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
