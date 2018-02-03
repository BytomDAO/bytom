package contract

import (
	"encoding/json"
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
)

// LockPubHash stores the information of LockWithPublicKeyHash contract
type LockPubHash struct {
	CommonInfo
	PublicKey string `json:"publickey"`
	PubKeyInfo
}

// DecodeLockPubHash unmarshal JSON-encoded data of contract action
func DecodeLockPubHash(data []byte) (ContractAction, error) {
	a := new(LockPubHash)
	err := json.Unmarshal(data, a)
	return a, err
}

// BuildContractReq create new ContractReq which contain contract's name and arguments
func (a *LockPubHash) BuildContractReq(contractName string) (*ContractReq, error) {
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
func (a *LockPubHash) Build() (*string, error) {
	var buildReqStr string

	if a.Alias {
		buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, a.OutputID, a.AssetInfo, a.Amount, a.AccountInfo, a.BtmGas, a.AccountInfo)
	} else {
		buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, a.OutputID, a.AssetInfo, a.Amount, a.AccountInfo, a.BtmGas, a.AccountInfo)
	}

	return &buildReqStr, nil
}

// AddArgs add the parameters for contract
func (a *LockPubHash) AddArgs(tpl *txbuilder.Template) (*txbuilder.Template, error) {
	var err error
	pubInfo := NewPubKeyInfo(a.RootPubKey, a.Path)
	paramInfo := NewParamInfo([]string{a.PublicKey}, []PubKeyInfo{pubInfo}, nil)

	if tpl, err = addParamArgs(tpl, paramInfo); err != nil {
		return nil, err
	}

	return tpl, nil
}
