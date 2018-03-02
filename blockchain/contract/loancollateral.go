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
func (a *LoanCollateral) Build() (buildReqStr string, err error) {
	switch a.Selector {
	case ClauseRepay:
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildInlineProgReqFmtByAlias, a.OutputID, a.AccountInfo,
				a.InnerAssetInfo, a.InnerAmount, a.InnerProgram,
				a.AssetInfo, a.Amount, a.ControlProgram,
				a.InnerAssetInfo, a.InnerAmount, a.InnerAccountInfo,
				a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildInlineProgReqFmt, a.OutputID, a.AccountInfo,
				a.InnerAssetInfo, a.InnerAmount, a.InnerProgram,
				a.AssetInfo, a.Amount, a.ControlProgram,
				a.InnerAssetInfo, a.InnerAmount, a.InnerAccountInfo,
				a.BtmGas, a.AccountInfo)
		}
	case ClauseDefault:
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, a.OutputID, a.AccountInfo, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, a.OutputID, a.AccountInfo, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		}
	default:
		err = errors.WithDetailf(ErrBadClause, "selected clause [%v] error, contract LoanCollateral's clause must in set:[%v, %v]",
			a.Selector, ClauseRepay, ClauseDefault)
	}

	return
}

// AddArgs add the parameters for contract
func (a *LoanCollateral) AddArgs(tpl *txbuilder.Template) (err error) {
	switch a.Selector {
	case ClauseRepay, ClauseDefault:
		err = addDataArgs(tpl, []string{a.Selector})
	default:
		err = errors.WithDetailf(ErrBadClause, "the selector[%s] for contract LoanCollateral is wrong!", a.Selector)
	}

	return
}
