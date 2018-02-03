package commands

import (
	"github.com/bytom/blockchain/contract"
	"github.com/bytom/errors"
)

// BuildReq build the request for contact
func BuildReq(contractName string, args []string, alias bool, btmGas string) (*contract.ContractReq, error) {
	var req *contract.ContractReq
	var err error

	switch contractName {
	case "LockWithPublicKey":
		contr := NewLockPubKey(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "LockWithMultiSig":
		contr := NewLockMultiSig(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "LockWithPublicKeyHash":
		contr := NewLockPubHash(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "RevealPreimage":
		contr := NewRevealPreimage(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "TradeOffer":
		contr := NewTradeOffer(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "Escrow":
		contr := NewEscrow(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "LoanCollateral":
		contr := NewLoanCollateral(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	case "CallOption":
		contr := NewCallOption(args, alias, btmGas)
		req, err = contr.BuildContractReq(contractName)
	default:
		err = errors.New("Invalid contract!")
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewLockPubKey create the contract object for LockWithPublicKey
func NewLockPubKey(args []string, alias bool, btmGas string) *contract.LockPubKey {
	var contr contract.LockPubKey

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.RootPubKey = args[4]
	contr.Path = []string{args[5], args[6]}

	return &contr
}

// NewLockMultiSig create the contract object for LockWithMultiSig
func NewLockMultiSig(args []string, alias bool, btmGas string) *contract.LockMultiSig {
	var contr contract.LockMultiSig

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas

	pubInfo1 := contract.NewPubKeyInfo(args[4], []string{args[5], args[6]})
	pubInfo2 := contract.NewPubKeyInfo(args[7], []string{args[8], args[9]})
	contr.PubKeys = []contract.PubKeyInfo{pubInfo1, pubInfo2}

	return &contr
}

// NewLockPubHash create the contract object for LockWithPublicKeyHash
func NewLockPubHash(args []string, alias bool, btmGas string) *contract.LockPubHash {
	var contr contract.LockPubHash

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.PublicKey = args[4]
	contr.RootPubKey = args[5]
	contr.Path = []string{args[6], args[7]}

	return &contr
}

// NewRevealPreimage create the contract object for RevealPreimage
func NewRevealPreimage(args []string, alias bool, btmGas string) *contract.RevealPreimage {
	var contr contract.RevealPreimage

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.Value = args[4]

	return &contr
}

// NewTradeOffer create the contract object for TradeOffer
func NewTradeOffer(args []string, alias bool, btmGas string) *contract.TradeOffer {
	var contr contract.TradeOffer

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.Selector = args[4]

	if contr.Selector == contract.ClauseTrade {
		contr.InnerAccountInfo = args[5]
		contr.InnerAssetInfo = args[6]
		contr.InnerAmount = args[7]
		contr.InnerProgram = args[8]
	} else if contr.Selector == contract.ClauseCancel {
		contr.RootPubKey = args[5]
		contr.Path = []string{args[6], args[7]}
	}

	return &contr
}

// NewEscrow create the contract object for Escrow
func NewEscrow(args []string, alias bool, btmGas string) *contract.Escrow {
	var contr contract.Escrow

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.Selector = args[4]
	contr.RootPubKey = args[5]
	contr.Path = []string{args[6], args[7]}
	contr.ControlProgram = args[8]

	return &contr
}

// NewLoanCollateral create the contract object for LoanCollateral
func NewLoanCollateral(args []string, alias bool, btmGas string) *contract.LoanCollateral {
	var contr contract.LoanCollateral

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.Selector = args[4]

	if contr.Selector == contract.ClauseRepay {
		contr.InnerAccountInfo = args[5]
		contr.InnerAssetInfo = args[6]
		contr.InnerAmount = args[7]
		contr.InnerProgram = args[8]
		contr.ControlProgram = args[9]
	} else if contr.Selector == contract.ClauseDefault {
		contr.ControlProgram = args[5]
	}

	return &contr
}

// NewCallOption create the contract object for CallOption
func NewCallOption(args []string, alias bool, btmGas string) *contract.CallOption {
	var contr contract.CallOption

	contr.OutputID = args[0]
	contr.AccountInfo = args[1]
	contr.AssetInfo = args[2]
	contr.Amount = args[3]
	contr.Alias = alias
	contr.BtmGas = btmGas
	contr.Selector = args[4]

	if contr.Selector == contract.ClauseExercise {
		contr.InnerAccountInfo = args[5]
		contr.InnerAssetInfo = args[6]
		contr.InnerAmount = args[7]
		contr.InnerProgram = args[8]
		contr.RootPubKey = args[9]
		contr.Path = []string{args[10], args[11]}
	} else if contr.Selector == contract.ClauseExpire {
		contr.ControlProgram = args[5]
	}

	return &contr
}
