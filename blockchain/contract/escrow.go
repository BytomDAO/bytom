package contract

import (
	"encoding/json"
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseApprove is the contract Escrow's clause approve
	ClauseApprove string = "00000000"
	//ClauseReject is the contract Escrow's clause reject
	ClauseReject string = "1b000000"
	//EscrowEnding is the contract Escrow's clause ending
	EscrowEnding string = "2a000000"
)

// Escrow stores the information of Escrow contract.
type Escrow struct {
	CommonInfo
	Selector       string `json:"selector"`
	ControlProgram string `json:"control_program"`
	PubKeyInfo
}

// DecodeEscrow unmarshal JSON-encoded data of contract action
func DecodeEscrow(data []byte) (ContractAction, error) {
	a := new(Escrow)
	err := json.Unmarshal(data, a)
	return a, err
}

// BuildContractReq create new ContractReq which contain contract's name and arguments
func (a *Escrow) BuildContractReq(contractName string) (*ContractReq, error) {
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
func (a *Escrow) Build() (*string, error) {
	var buildReqStr string
	var buf string

	if a.Selector == ClauseApprove || a.Selector == ClauseReject {
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		}
	} else {
		if a.Selector == EscrowEnding {
			buf = fmt.Sprintf("no clause was selected in this program, ending exit")
		} else {
			buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", a.Selector, ClauseApprove, ClauseReject, EscrowEnding)
		}

		err := errors.New(buf)
		return nil, err
	}

	return &buildReqStr, nil
}

// AddArgs add the parameters for contract
func (a *Escrow) AddArgs(tpl *txbuilder.Template) (*txbuilder.Template, error) {
	var err error

	if a.Selector == ClauseApprove || a.Selector == ClauseReject {
		pubInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
		paramInfo := NewParamInfo(nil, []PubKeyInfo{pubInfo}, []string{a.Selector})

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
