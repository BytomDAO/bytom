package contract

import (
	"encoding/hex"
	"fmt"

	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
)

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
			fmt.Println(usage + " <clauseSelector> (<innerAssetID|alias> <innerAmount> <innerAccountID|alias> <innerProgram>) " +
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
			fmt.Println(usage + " <clauseSelector> (<innerAssetID|alias> <innerAmount> <innerAccountID|alias> <innerProgram> <controlProgram>) " +
				"| (<controlProgram>) [flags]\n")
			return false
		}
	case "CallOption":
		if !(len(args) == count+2 || len(args) == count+8) {
			fmt.Println(usage + " <clauseSelector> (<innerAssetID|alias> <innerAmount> <innerAccountID|alias> <innerProgram> <rootPub> <path1> <path2>) " +
				"| (<controlProgram>) [flags]\n")
			return false
		}
	default:
		fmt.Println("Invalid contract template name")
		return false
	}

	return true
}

func reconstructTpl(tpl *txbuilder.Template, si *txbuilder.SigningInstruction) *txbuilder.Template {
	length := len(tpl.SigningInstructions)
	if length <= 0 {
		length = 1
		tpl.SigningInstructions = append(tpl.SigningInstructions, si)
		tpl.SigningInstructions[length-1].Position = 0
	} else {
		tpl.SigningInstructions[0] = si
	}

	return tpl
}

func addPublicKeyWitness(tpl *txbuilder.Template, rootPub string, path1 string, path2 string) (*txbuilder.Template, error) {
	var rootPubKey chainkd.XPub
	var path []chainjson.HexBytes
	var totalRoot []chainkd.XPub
	var totalPath [][]chainjson.HexBytes
	var si txbuilder.SigningInstruction

	root, err := hex.DecodeString(rootPub)
	if err != nil {
		return nil, err
	}
	copy(rootPubKey[:], root[:])

	p1, err := hex.DecodeString(path1)
	if err != nil {
		return nil, err
	}

	p2, err := hex.DecodeString(path2)
	if err != nil {
		return nil, err
	}

	path = append(path, p1)
	path = append(path, p2)
	totalRoot = append(totalRoot, rootPubKey)
	totalPath = append(totalPath, path)

	err = si.AddRawTxSigWitness(totalRoot, totalPath, 1)
	if err != nil {
		return nil, err
	}

	tpl = reconstructTpl(tpl, &si)

	return tpl, nil
}

func addMultiSigWitness(tpl *txbuilder.Template, rootPub string, path1 string, path2 string, rootPub1 string, path11 string, path12 string) (*txbuilder.Template, error) {
	var rootPubKey chainkd.XPub
	var firstPath []chainjson.HexBytes
	var secondPath []chainjson.HexBytes
	var totalRoot []chainkd.XPub
	var totalPath [][]chainjson.HexBytes
	var si txbuilder.SigningInstruction

	//add the first arguments
	root, err := hex.DecodeString(rootPub)
	if err != nil {
		return nil, err
	}
	copy(rootPubKey[:], root[:])

	p1, err := hex.DecodeString(path1)
	if err != nil {
		return nil, err
	}

	p2, err := hex.DecodeString(path2)
	if err != nil {
		return nil, err
	}

	firstPath = append(firstPath, p1)
	firstPath = append(firstPath, p2)

	totalRoot = append(totalRoot, rootPubKey)
	totalPath = append(totalPath, firstPath)

	//add the second arguments
	root, err = hex.DecodeString(rootPub1)
	if err != nil {
		return nil, err
	}
	copy(rootPubKey[:], root[:])

	p1, err = hex.DecodeString(path11)
	if err != nil {
		return nil, err
	}

	p2, err = hex.DecodeString(path12)
	if err != nil {
		return nil, err
	}

	secondPath = append(secondPath, p1)
	secondPath = append(secondPath, p2)
	totalRoot = append(totalRoot, rootPubKey)
	totalPath = append(totalPath, secondPath)

	err = si.AddRawTxSigWitness(totalRoot, totalPath, 2)
	if err != nil {
		return nil, err
	}

	tpl = reconstructTpl(tpl, &si)

	return tpl, nil
}

func addPublicKeyHashWitness(tpl *txbuilder.Template, pubKey string, rootPub string, path1 string, path2 string) (*txbuilder.Template, error) {
	var data []chainjson.HexBytes
	var rootPubKey chainkd.XPub
	var path []chainjson.HexBytes
	var totalRoot []chainkd.XPub
	var totalPath [][]chainjson.HexBytes
	var si txbuilder.SigningInstruction

	pubkey, err := hex.DecodeString(pubKey)
	if err != nil {
		return nil, err
	}
	data = append(data, pubkey)
	si.AddDataWitness(data)

	root, err := hex.DecodeString(rootPub)
	if err != nil {
		return nil, err
	}
	copy(rootPubKey[:], root[:])

	p1, err := hex.DecodeString(path1)
	if err != nil {
		return nil, err
	}

	p2, err := hex.DecodeString(path2)
	if err != nil {
		return nil, err
	}

	path = append(path, p1)
	path = append(path, p2)
	totalRoot = append(totalRoot, rootPubKey)
	totalPath = append(totalPath, path)

	err = si.AddRawTxSigWitness(totalRoot, totalPath, 1)
	if err != nil {
		return nil, err
	}

	tpl = reconstructTpl(tpl, &si)

	return tpl, nil
}

func addValueWitness(tpl *txbuilder.Template, value string) (*txbuilder.Template, error) {
	var data []chainjson.HexBytes
	var si txbuilder.SigningInstruction

	str, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	data = append(data, str)
	si.AddDataWitness(data)

	tpl = reconstructTpl(tpl, &si)

	return tpl, nil
}

func addPubValueWitness(tpl *txbuilder.Template, rootPub string, path1 string, path2 string, selector string) (*txbuilder.Template, error) {
	var data []chainjson.HexBytes
	var rootPubKey chainkd.XPub
	var path []chainjson.HexBytes
	var totalRoot []chainkd.XPub
	var totalPath [][]chainjson.HexBytes
	var si txbuilder.SigningInstruction

	root, err := hex.DecodeString(rootPub)
	if err != nil {
		return nil, err
	}
	copy(rootPubKey[:], root[:])

	p1, err := hex.DecodeString(path1)
	if err != nil {
		return nil, err
	}

	p2, err := hex.DecodeString(path2)
	if err != nil {
		return nil, err
	}

	path = append(path, p1)
	path = append(path, p2)
	totalRoot = append(totalRoot, rootPubKey)
	totalPath = append(totalPath, path)

	err = si.AddRawTxSigWitness(totalRoot, totalPath, 1)
	if err != nil {
		return nil, err
	}

	str, err := hex.DecodeString(selector)
	if err != nil {
		return nil, err
	}
	data = append(data, str)
	si.AddDataWitness(data)

	tpl = reconstructTpl(tpl, &si)

	return tpl, nil
}
