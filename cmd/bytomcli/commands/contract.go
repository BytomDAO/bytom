package commands

import (
	"github.com/bytom/blockchain/contract"
	"github.com/bytom/errors"
)

// CheckContractArgs check the number of arguments for template contracts
func CheckContractArgs(contractName string, args []string, count int, usage string) (err error) {
	switch contractName {
	case "LockWithPublicKey":
		if len(args) != count+3 {
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <rootPub> <path1> <path2> [flags]\n", usage)
		}
	case "LockWithMultiSig":
		if len(args) != count+6 {
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <rootPub1> <path11> <path12> <rootPub2> <path21> <path22> [flags]\n", usage)
		}
	case "LockWithPublicKeyHash":
		if len(args) != count+4 {
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <pubKey> <rootPub> <path1> <path2> [flags]\n", usage)
		}
	case "RevealPreimage":
		if len(args) != count+1 {
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <value> [flags]\n", usage)
		}
	case "TradeOffer":
		switch {
		case len(args) <= count:
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram>) | (<rootPub> <path1> <path2>) [flags]\n", usage)
		case args[count] == contract.ClauseTrade:
			if len(args) != count+5 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> [flags]\n", usage)
			}
		case args[count] == contract.ClauseCancel:
			if len(args) != count+4 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <rootPub> <path1> <path2> [flags]\n", usage)
			}
		case args[count] == contract.TradeOfferEnding:
			err = errors.WithDetailf(contract.ErrBadArguments, "Clause ending was selected in contract %s, ending exit!", contractName)
		default:
			err = errors.WithDetailf(contract.ErrBadArguments, "selected clause [%s] error, contract %s's clause must in set:[%s, %s, %s]",
				args[count], contractName, contract.ClauseTrade, contract.ClauseCancel, contract.TradeOfferEnding)
		}
	case "Escrow":
		switch {
		case len(args) <= count:
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <rootPub> <path1> <path2> <controlProgram> [flags]\n", usage)
		case args[count] == contract.ClauseApprove || args[count] == contract.ClauseReject:
			if len(args) != count+5 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <rootPub> <path1> <path2> <controlProgram> [flags]\n", usage)
			}
		case args[count] == contract.EscrowEnding:
			err = errors.WithDetailf(contract.ErrBadArguments, "Clause ending was selected in contract %s, ending exit!", contractName)
		default:
			err = errors.WithDetailf(contract.ErrBadArguments, "selected clause [%s] error, contract %s's clause must in set:[%s, %s, %s]",
				args[count], contractName, contract.ClauseApprove, contract.ClauseReject, contract.EscrowEnding)
		}
	case "LoanCollateral":
		switch {
		case len(args) <= count:
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <controlProgram>) | (<controlProgram>) [flags]\n", usage)
		case args[count] == contract.ClauseRepay:
			if len(args) != count+6 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <controlProgram> [flags]\n", usage)
			}
		case args[count] == contract.ClauseDefault:
			if len(args) != count+2 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <controlProgram> [flags]\n", usage)
			}
		case args[count] == contract.LoanCollateralEnding:
			err = errors.WithDetailf(contract.ErrBadArguments, "Clause ending was selected in contract %s, ending exit!", contractName)
		default:
			err = errors.WithDetailf(contract.ErrBadArguments, "selected clause [%s] error, contract %s's clause must in set:[%s, %s, %s]",
				args[count], contractName, contract.ClauseRepay, contract.ClauseDefault, contract.LoanCollateralEnding)
		}
	case "CallOption":
		switch {
		case len(args) <= count:
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <rootPub> <path1> <path2>) | (<controlProgram>) [flags]\n", usage)
		case args[count] == contract.ClauseExercise:
			if len(args) != count+8 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <rootPub> <path1> <path2> [flags]\n", usage)
			}
		case args[count] == contract.ClauseExpire:
			if len(args) != count+2 {
				err = errors.WithDetailf(contract.ErrBadArguments, "%s <clauseSelector> <controlProgram> [flags]\n", usage)
			}
		case args[count] == contract.CallOptionEnding:
			err = errors.WithDetailf(contract.ErrBadArguments, "Clause ending was selected in contract %s, ending exit!", contractName)
		default:
			err = errors.WithDetailf(contract.ErrBadArguments, "selected clause [%s] error, contract %s's clause must in set:[%s, %s, %s]",
				args[count], contractName, contract.ClauseExercise, contract.ClauseExpire, contract.CallOptionEnding)
		}
	default:
		err = errors.WithDetailf(contract.ErrBadArguments, "Invalid contract template name:%s", contractName)
	}

	return
}

// BuildReq build the request for contact
func BuildReq(contractName string, args []string, alias bool, btmGas string) (req *contract.ContractReq, err error) {
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
	}

	return
}

// NewLockPubKey create the contract object for LockWithPublicKey
func NewLockPubKey(args []string, alias bool, btmGas string) *contract.LockPubKey {
	return &contract.LockPubKey{
		CommonInfo: contract.CommonInfo{
			OutputID:    args[0],
			AccountInfo: args[1],
			AssetInfo:   args[2],
			Amount:      args[3],
			Alias:       alias,
			BtmGas:      btmGas,
		},
		PubKeyInfo: contract.PubKeyInfo{
			RootPubKey: args[4],
			Path:       []string{args[5], args[6]},
		},
	}
}

// NewLockMultiSig create the contract object for LockWithMultiSig
func NewLockMultiSig(args []string, alias bool, btmGas string) *contract.LockMultiSig {
	pubInfo1 := contract.NewPubKeyInfo(args[4], []string{args[5], args[6]})
	pubInfo2 := contract.NewPubKeyInfo(args[7], []string{args[8], args[9]})

	return &contract.LockMultiSig{
		CommonInfo: contract.CommonInfo{
			OutputID:    args[0],
			AccountInfo: args[1],
			AssetInfo:   args[2],
			Amount:      args[3],
			Alias:       alias,
			BtmGas:      btmGas,
		},
		PubKeys: []contract.PubKeyInfo{pubInfo1, pubInfo2},
	}
}

// NewLockPubHash create the contract object for LockWithPublicKeyHash
func NewLockPubHash(args []string, alias bool, btmGas string) *contract.LockPubHash {
	return &contract.LockPubHash{
		CommonInfo: contract.CommonInfo{
			OutputID:    args[0],
			AccountInfo: args[1],
			AssetInfo:   args[2],
			Amount:      args[3],
			Alias:       alias,
			BtmGas:      btmGas,
		},
		PublicKey: args[4],
		PubKeyInfo: contract.PubKeyInfo{
			RootPubKey: args[5],
			Path:       []string{args[6], args[7]},
		},
	}
}

// NewRevealPreimage create the contract object for RevealPreimage
func NewRevealPreimage(args []string, alias bool, btmGas string) *contract.RevealPreimage {
	return &contract.RevealPreimage{
		CommonInfo: contract.CommonInfo{
			OutputID:    args[0],
			AccountInfo: args[1],
			AssetInfo:   args[2],
			Amount:      args[3],
			Alias:       alias,
			BtmGas:      btmGas,
		},
		Value: args[4],
	}
}

// NewTradeOffer create the contract object for TradeOffer
func NewTradeOffer(args []string, alias bool, btmGas string) *contract.TradeOffer {
	selector := args[4]
	switch selector {
	case contract.ClauseTrade:
		return &contract.TradeOffer{
			CommonInfo: contract.CommonInfo{
				OutputID:    args[0],
				AccountInfo: args[1],
				AssetInfo:   args[2],
				Amount:      args[3],
				Alias:       alias,
				BtmGas:      btmGas,
			},
			Selector: args[4],
			PaymentInfo: contract.PaymentInfo{
				InnerAccountInfo: args[5],
				InnerAssetInfo:   args[6],
				InnerAmount:      args[7],
				InnerProgram:     args[8],
			},
		}
	case contract.ClauseCancel:
		return &contract.TradeOffer{
			CommonInfo: contract.CommonInfo{
				OutputID:    args[0],
				AccountInfo: args[1],
				AssetInfo:   args[2],
				Amount:      args[3],
				Alias:       alias,
				BtmGas:      btmGas,
			},
			Selector: args[4],
			PubKeyInfo: contract.PubKeyInfo{
				RootPubKey: args[5],
				Path:       []string{args[6], args[7]},
			},
		}
	default:
		return nil
	}
}

// NewEscrow create the contract object for Escrow
func NewEscrow(args []string, alias bool, btmGas string) *contract.Escrow {
	return &contract.Escrow{
		CommonInfo: contract.CommonInfo{
			OutputID:    args[0],
			AccountInfo: args[1],
			AssetInfo:   args[2],
			Amount:      args[3],
			Alias:       alias,
			BtmGas:      btmGas,
		},
		Selector: args[4],
		PubKeyInfo: contract.PubKeyInfo{
			RootPubKey: args[5],
			Path:       []string{args[6], args[7]},
		},
		ControlProgram: args[8],
	}
}

// NewLoanCollateral create the contract object for LoanCollateral
func NewLoanCollateral(args []string, alias bool, btmGas string) *contract.LoanCollateral {
	selector := args[4]
	switch selector {
	case contract.ClauseRepay:
		return &contract.LoanCollateral{
			CommonInfo: contract.CommonInfo{
				OutputID:    args[0],
				AccountInfo: args[1],
				AssetInfo:   args[2],
				Amount:      args[3],
				Alias:       alias,
				BtmGas:      btmGas,
			},
			Selector: args[4],
			PaymentInfo: contract.PaymentInfo{
				InnerAccountInfo: args[5],
				InnerAssetInfo:   args[6],
				InnerAmount:      args[7],
				InnerProgram:     args[8],
			},
			ControlProgram: args[9],
		}
	case contract.ClauseDefault:
		return &contract.LoanCollateral{
			CommonInfo: contract.CommonInfo{
				OutputID:    args[0],
				AccountInfo: args[1],
				AssetInfo:   args[2],
				Amount:      args[3],
				Alias:       alias,
				BtmGas:      btmGas,
			},
			Selector:       args[4],
			ControlProgram: args[5],
		}
	default:
		return nil
	}
}

// NewCallOption create the contract object for CallOption
func NewCallOption(args []string, alias bool, btmGas string) *contract.CallOption {
	selector := args[4]
	switch selector {
	case contract.ClauseExercise:
		return &contract.CallOption{
			CommonInfo: contract.CommonInfo{
				OutputID:    args[0],
				AccountInfo: args[1],
				AssetInfo:   args[2],
				Amount:      args[3],
				Alias:       alias,
				BtmGas:      btmGas,
			},
			Selector: args[4],
			PaymentInfo: contract.PaymentInfo{
				InnerAccountInfo: args[5],
				InnerAssetInfo:   args[6],
				InnerAmount:      args[7],
				InnerProgram:     args[8],
			},
			PubKeyInfo: contract.PubKeyInfo{
				RootPubKey: args[9],
				Path:       []string{args[10], args[11]},
			},
		}
	case contract.ClauseExpire:
		return &contract.CallOption{
			CommonInfo: contract.CommonInfo{
				OutputID:    args[0],
				AccountInfo: args[1],
				AssetInfo:   args[2],
				Amount:      args[3],
				Alias:       alias,
				BtmGas:      btmGas,
			},
			Selector:       args[4],
			ControlProgram: args[5],
		}
	default:
		return nil
	}
}
