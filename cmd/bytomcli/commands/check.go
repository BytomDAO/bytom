package commands

import (
	"github.com/bytom/blockchain/contract"
	"github.com/bytom/errors"
)

// CheckContractArgs check the number of arguments for template contracts
func CheckContractArgs(contractName string, args []string, count int, usage string) error {
	var err error
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
			err = errors.WithDetailf(contract.ErrBadArguments, "%s <value> [flags]\n")
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

	if err != nil {
		return err
	}

	return nil
}
