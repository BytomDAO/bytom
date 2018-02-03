package contract

import (
	stdjson "encoding/json"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

// ContractReq stores the information of contract
type ContractReq struct {
	ContractName string             `json:"contract_name"`
	ContractArgs stdjson.RawMessage `json:"contract_args"`
}

// ContractAction represents the operation action for contract
type ContractAction interface {
	Build() (*string, error)
	AddArgs(tpl *txbuilder.Template) (*txbuilder.Template, error)
}

// ContractDecoder generalize contract objects into an interface
func (a *ContractReq) ContractDecoder() (ContractAction, error) {
	var act ContractAction
	var err error

	switch a.ContractName {
	case "LockWithPublicKey":
		act, err = DecodeLockPubKey(a.ContractArgs)
	case "LockWithMultiSig":
		act, err = DecodeLockMultiSig(a.ContractArgs)
	case "LockWithPublicKeyHash":
		act, err = DecodeLockPubHash(a.ContractArgs)
	case "RevealPreimage":
		act, err = DecodeRevealPreimage(a.ContractArgs)
	case "TradeOffer":
		act, err = DecodeTradeOffer(a.ContractArgs)
	case "Escrow":
		act, err = DecodeEscrow(a.ContractArgs)
	case "LoanCollateral":
		act, err = DecodeLoanCollateral(a.ContractArgs)
	case "CallOption":
		act, err = DecodeCallOption(a.ContractArgs)
	default:
		err = errors.New("Invalid contract!")
	}

	if err != nil {
		return nil, err
	}

	return act, nil
}
