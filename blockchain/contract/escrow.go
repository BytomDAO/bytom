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
func (a *Escrow) Build() (buildReqStr string, err error) {
	switch a.Selector {
	case ClauseApprove, ClauseReject:
		if a.Alias {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.ControlProgram, a.BtmGas, a.AccountInfo)
		}
	default:
		err = errors.WithDetailf(ErrBadClause, "selected clause [%v] error, contract Escrow's clause must in set:[%v, %v]",
			a.Selector, ClauseApprove, ClauseReject)
	}

	return
}

// AddArgs add the parameters for contract
func (a *Escrow) AddArgs(tpl *txbuilder.Template) (err error) {
	switch a.Selector {
	case ClauseApprove, ClauseReject:
		pubInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
		paramInfo := NewParamInfo(nil, []PubKeyInfo{pubInfo}, []string{a.Selector})
		err = addParamArgs(tpl, paramInfo)
	default:
		err = errors.WithDetailf(ErrBadClause, "the selector[%s] for contract Escrow is wrong!", a.Selector)
	}

	return
}
