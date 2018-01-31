package commands

import "fmt"

// CheckContractArgs check the number of arguments for template contracts
func CheckContractArgs(contractName string, args []string, count int, usage string) bool {
	switch contractName {
	case "LockWithPublicKey":
		if len(args) != count+3 {
			fmt.Println(usage + " <rootPub> <path1> <path2> [flags]\n")
			return false
		}
	case "LockWithMultiSig":
		if len(args) != count+6 {
			fmt.Println(usage + " <rootPub1> <path11> <path12> <rootPub2> <path21> <path22> [flags]\n")
			return false
		}
	case "LockWithPublicKeyHash":
		if len(args) != count+4 {
			fmt.Println(usage + " <pubKey> <rootPub> <path1> <path2> [flags]\n")
			return false
		}
	case "RevealPreimage":
		if len(args) != count+1 {
			fmt.Println(usage + " <value> [flags]\n")
			return false
		}
	case "TradeOffer":
		if !(len(args) == count+4 || len(args) == count+5) {
			fmt.Println(usage + " <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram>) " +
				"| (<rootPub> <path1> <path2>) [flags]\n")
			return false
		}
	case "Escrow":
		if len(args) != count+5 {
			fmt.Println(usage + " <clauseSelector> <rootPub> <path1> <path2> <controlProgram> [flags]\n")
			return false
		}
	case "LoanCollateral":
		if !(len(args) == count+2 || len(args) == count+6) {
			fmt.Println(usage + " <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <controlProgram>) " +
				"| (<controlProgram>) [flags]\n")
			return false
		}
	case "CallOption":
		if !(len(args) == count+2 || len(args) == count+8) {
			fmt.Println(usage + " <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> <rootPub> <path1> <path2>) " +
				"| (<controlProgram>) [flags]\n")
			return false
		}
	default:
		fmt.Println("Invalid contract template name")
		return false
	}

	return true
}
