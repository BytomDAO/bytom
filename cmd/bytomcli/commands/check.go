package commands

import (
	"fmt"
	"github.com/bytom/errors"
)

// CheckContractArgs check the number of arguments for template contracts
func CheckContractArgs(contractName string, args []string, count int, usage string) error {
	var buf string

	switch contractName {
	case "LockWithPublicKey":
		if len(args) != count+3 {
			buf = fmt.Sprintf("%s <rootPub> <path1> <path2> [flags]\n", usage)
		}
	case "LockWithMultiSig":
		if len(args) != count+6 {
			buf = fmt.Sprintf("%s <rootPub1> <path11> <path12> <rootPub2> <path21> <path22> [flags]\n", usage)
		}
	case "LockWithPublicKeyHash":
		if len(args) != count+4 {
			buf = fmt.Sprintf("%s <pubKey> <rootPub> <path1> <path2> [flags]\n", usage)
		}
	case "RevealPreimage":
		if len(args) != count+1 {
			buf = fmt.Sprintf("%s <value> [flags]\n")
		}
	case "TradeOffer":
		if !(len(args) == count+4 || len(args) == count+5) {
			buf = fmt.Sprintf("%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram>) | (<rootPub> <path1> <path2>) [flags]\n", usage)
		}
	case "Escrow":
		if len(args) != count+5 {
			buf = fmt.Sprintf("%s <clauseSelector> <rootPub> <path1> <path2> <controlProgram> [flags]\n", usage)
		}
	case "LoanCollateral":
		if !(len(args) == count+2 || len(args) == count+6) {
			buf = fmt.Sprintf("%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <controlProgram>) | (<controlProgram>) [flags]\n", usage)
		}
	case "CallOption":
		if !(len(args) == count+2 || len(args) == count+8) {
			buf = fmt.Sprintf("%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <rootPub> <path1> <path2>) | (<controlProgram>) [flags]\n", usage)
		}
	default:
		buf = fmt.Sprintf("Invalid contract template name:%s", contractName)
	}

	if buf != "" {
		err := errors.New(buf)
		return err
	}

	return nil
}
