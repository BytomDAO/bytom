package contract

import (
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const (
	//ClauseTrade is the contract TradeOffer's clause trade
	ClauseTrade string = "00000000"
	//ClauseCancel is the contract TradeOffer's clause cancel
	ClauseCancel string = "13000000"
	//ClauseApprove is the contract Escrow's clause approve
	ClauseApprove string = "00000000"
	//ClauseReject is the contract Escrow's clause reject
	ClauseReject string = "1b000000"
	//ClauseRepay is the contract LoanCollateral's clause repay
	ClauseRepay string = "00000000"
	//ClauseDefault is the contract LoanCollateral's clause default
	ClauseDefault string = "1c000000"
	//ClauseExercise is the contract CallOption's clause exercise
	ClauseExercise string = "00000000"
	//ClauseExpire is the contract CallOption's clause expire
	ClauseExpire string = "22000000"
)

// AcctRecv is the contract contains only one statement with "unlock value"
var buildAcctRecvReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildAcctRecvReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"}
	]}`

// ProgRecv is the contract contains only one statement with "lock value with program"
var buildProgRecvReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildProgRecvReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"}
	]}`

// InlineAcct is the contract contains only two statement with "lock payment with program" and "unlock value"
var buildInlineAcctReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "control_account", "asset_id": "%s", "amount": %s, "account_id": "%s"}
	]}`

var buildInlineAcctReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"},
		{"type": "control_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"}
	]}`

// InlineAcct is the contract contains only two statement with "lock payment with program" and "lock value with control_program"
var buildInlineProgReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildInlineProgReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "btm", "amount": %s, "account_alias": "%s"}
	]}`

// BuildContractTransaction generate the template contract transaction actions for buildTransaction
func BuildContractTransaction(args []string, minArgsCount int, alias bool, btmGas string) (string, error) {
	var buildReqStr string
	var selector string
	var innerAssetInfo string
	var innerAmount string
	var innerAccountInfo string
	var program string
	var controlProgram string
	var buf string
	var err error
	const emptyStr = ""

	if len(args) < 5 {
		buf = fmt.Sprintf("the number of basic arguments[%d] is less than requirements[%d]", len(args), minArgsCount)
		err = errors.New(buf)
		return emptyStr, err
	}

	contractName := args[0]
	outputID := args[1]
	accountInfo := args[2]
	assetInfo := args[3]
	amount := args[4]

	switch contractName {
	case "LockWithPublicKey":
		if alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		}
	case "LockWithMultiSig":
		if alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		}
	case "LockWithPublicKeyHash":
		if alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		}
	case "RevealPreimage":
		if alias {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		} else {
			buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		}
	case "TradeOffer":
		ClauseEnding := "1a000000"
		selector = args[minArgsCount]

		if selector == ClauseTrade {
			if len(args) != minArgsCount+5 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'trade' in contract 'TradeOffer' is not equal to 5", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			innerAssetInfo = args[minArgsCount+1]
			innerAmount = args[minArgsCount+2]
			innerAccountInfo = args[minArgsCount+3]
			program = args[minArgsCount+4]
			if alias {
				buildReqStr = fmt.Sprintf(buildInlineAcctReqFmtByAlias, outputID,
					innerAssetInfo, innerAmount, program,
					innerAssetInfo, innerAmount, innerAccountInfo,
					btmGas, accountInfo,
					assetInfo, amount, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildInlineAcctReqFmt, outputID,
					innerAssetInfo, innerAmount, program,
					innerAssetInfo, innerAmount, innerAccountInfo,
					btmGas, accountInfo,
					assetInfo, amount, accountInfo)
			}
		} else if selector == ClauseCancel {
			if len(args) != minArgsCount+4 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'cancel' in contract 'TradeOffer' is not equal to 4", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			if alias {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
			}
		} else {
			if selector == ClauseEnding {
				buf = fmt.Sprintf("no clause was selected in this program, ending exit")
			} else {
				buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseTrade, ClauseCancel, ClauseEnding)
			}

			err = errors.New(buf)
			return emptyStr, err
		}

	case "Escrow":
		ClauseEnding := "2a000000"
		selector = args[minArgsCount]

		if selector == ClauseApprove || selector == ClauseReject {
			if len(args) != minArgsCount+5 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'approve' or 'reject' in contract 'Escrow' is not equal to 5", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			controlProgram = args[minArgsCount+4]
			if alias {
				buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
			}
		} else {
			if selector == ClauseEnding {
				buf = fmt.Sprintf("no clause was selected in this program, ending exit")
			} else {
				buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseApprove, ClauseReject, ClauseEnding)
			}

			err = errors.New(buf)
			return emptyStr, err
		}

	case "LoanCollateral":
		ClauseEnding := "28000000"
		selector = args[minArgsCount]

		if selector == ClauseRepay {
			if len(args) != minArgsCount+6 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'repay' in contract 'LoanCollateral' is not equal to 6", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			innerAssetInfo = args[minArgsCount+1]
			innerAmount = args[minArgsCount+2]
			innerAccountInfo = args[minArgsCount+3]
			program = args[minArgsCount+4]
			controlProgram = args[minArgsCount+5]
			if alias {
				buildReqStr = fmt.Sprintf(buildInlineProgReqFmtByAlias, outputID,
					innerAssetInfo, innerAmount, program,
					assetInfo, amount, controlProgram,
					innerAssetInfo, innerAmount, innerAccountInfo,
					btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildInlineProgReqFmt, outputID,
					innerAssetInfo, innerAmount, program,
					assetInfo, amount, controlProgram,
					innerAssetInfo, innerAmount, innerAccountInfo,
					btmGas, accountInfo)
			}
		} else if selector == ClauseDefault {
			if len(args) != minArgsCount+2 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'default' in contract 'LoanCollateral' is not equal to 2", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			controlProgram = args[minArgsCount+1]
			if alias {
				buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
			}
		} else {
			if selector == ClauseEnding {
				buf = fmt.Sprintf("no clause was selected in this program, ending exit")
			} else {
				buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseRepay, ClauseDefault, ClauseEnding)
			}

			err = errors.New(buf)
			return emptyStr, err
		}

	case "CallOption":
		ClauseEnding := "2f000000"
		selector = args[minArgsCount]

		if selector == ClauseExercise {
			if len(args) != minArgsCount+8 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'exercise' in contract 'CallOption' is not equal to 8", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			innerAssetInfo = args[minArgsCount+1]
			innerAmount = args[minArgsCount+2]
			innerAccountInfo = args[minArgsCount+3]
			program = args[minArgsCount+4]
			if alias {
				buildReqStr = fmt.Sprintf(buildInlineAcctReqFmtByAlias, outputID,
					innerAssetInfo, innerAmount, program,
					innerAssetInfo, innerAmount, innerAccountInfo,
					btmGas, accountInfo,
					assetInfo, amount, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildInlineAcctReqFmt, outputID,
					innerAssetInfo, innerAmount, program,
					innerAssetInfo, innerAmount, innerAccountInfo,
					btmGas, accountInfo,
					assetInfo, amount, accountInfo)
			}
		} else if selector == ClauseExpire {
			if len(args) != minArgsCount+2 {
				buf = fmt.Sprintf("the number of arguments[%d] for clause 'expire' in contract 'CallOption' is not equal to 2", len(args)-minArgsCount)
				err = errors.New(buf)
				return emptyStr, err
			}

			controlProgram = args[minArgsCount+1]
			if alias {
				buildReqStr = fmt.Sprintf(buildProgRecvReqFmtByAlias, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
			} else {
				buildReqStr = fmt.Sprintf(buildProgRecvReqFmt, outputID, assetInfo, amount, controlProgram, btmGas, accountInfo)
			}
		} else {
			if selector == ClauseEnding {
				buf = fmt.Sprintf("no clause was selected in this program, ending exit")
			} else {
				buf = fmt.Sprintf("selected clause [%v] error, clause must in set:[%v, %v, %v]", selector, ClauseExercise, ClauseExpire, ClauseEnding)
			}

			err = errors.New(buf)
			return emptyStr, err
		}

	default:
		buf = fmt.Sprintf("Invalid contract template name")
		err = errors.New(buf)
		return emptyStr, err
	}

	return buildReqStr, nil
}

// AddContractArguments add the template contract arguments into txbuilder.Template
func AddContractArguments(tpl *txbuilder.Template, contractName string, contractArgs []string) (*txbuilder.Template, error) {
	var rootPub string
	var path1 string
	var path2 string
	var rootPub1 string
	var path11 string
	var path12 string
	var pubKey string
	var value string
	var selector string
	var buf string
	var err error
	var errArgs error

	//add paraments for Witness
	switch contractName {
	case "LockWithPublicKey":
		if len(contractArgs) != 3 {
			buf = fmt.Sprintf("the arguments of contract 'LockWithPublicKey' is not equal to 3")
			err = errors.New(buf)
			return nil, err
		}

		rootPub = contractArgs[0]
		path1 = contractArgs[1]
		path2 = contractArgs[2]

		tpl, errArgs = addPublicKeyWitness(tpl, rootPub, path1, path2)
	case "LockWithMultiSig":
		if len(contractArgs) != 6 {
			buf = fmt.Sprintf("the arguments of contract 'LockWithMultiSig' is not equal to 6")
			err = errors.New(buf)
			return nil, err
		}

		rootPub = contractArgs[0]
		path1 = contractArgs[1]
		path2 = contractArgs[2]
		rootPub1 = contractArgs[3]
		path11 = contractArgs[4]
		path12 = contractArgs[5]

		tpl, errArgs = addMultiSigWitness(tpl, rootPub, path1, path2, rootPub1, path11, path12)
	case "LockWithPublicKeyHash":
		if len(contractArgs) != 4 {
			buf = fmt.Sprintf("the arguments of contract 'LockWithPublicKeyHash' is not equal to 4")
			err = errors.New(buf)
			return nil, err
		}

		pubKey = contractArgs[0]
		rootPub = contractArgs[1]
		path1 = contractArgs[2]
		path2 = contractArgs[3]

		tpl, errArgs = addPublicKeyHashWitness(tpl, pubKey, rootPub, path1, path2)
	case "RevealPreimage":
		if len(contractArgs) != 1 {
			buf = fmt.Sprintf("the arguments of contract 'RevealPreimage' is not equal to 1")
			err = errors.New(buf)
			return nil, err
		}

		value = contractArgs[0]

		tpl, errArgs = addValueWitness(tpl, value)
	case "TradeOffer":
		if !(len(contractArgs) == 4 || len(contractArgs) == 5) {
			buf = fmt.Sprintf("the arguments of contract 'TradeOffer' is not equal to 4 or 5")
			err = errors.New(buf)
			return nil, err
		}

		selector = contractArgs[0]

		if selector == ClauseTrade {
			tpl, errArgs = addValueWitness(tpl, selector)
		} else if selector == ClauseCancel {
			rootPub = contractArgs[1]
			path1 = contractArgs[2]
			path2 = contractArgs[3]

			tpl, errArgs = addPubValueWitness(tpl, rootPub, path1, path2, selector)
		}
	case "Escrow":
		if len(contractArgs) != 5 {
			buf = fmt.Sprintf("the arguments of contract 'Escrow' is not equal to 5")
			err = errors.New(buf)
			return nil, err
		}

		selector = contractArgs[0]
		rootPub = contractArgs[1]
		path1 = contractArgs[2]
		path2 = contractArgs[3]

		tpl, errArgs = addPubValueWitness(tpl, rootPub, path1, path2, selector)
	case "LoanCollateral":
		if !(len(contractArgs) == 2 || len(contractArgs) == 6) {
			buf = fmt.Sprintf("the arguments of contract 'LoanCollateral' is not equal to 2 or 6")
			err = errors.New(buf)
			return nil, err
		}

		selector = contractArgs[0]
		tpl, errArgs = addValueWitness(tpl, selector)

	case "CallOption":
		if !(len(contractArgs) == 2 || len(contractArgs) == 8) {
			buf = fmt.Sprintf("the arguments of contract 'CallOption' is not equal to 2 or 8")
			err = errors.New(buf)
			return nil, err
		}

		selector = contractArgs[0]

		if selector == ClauseExercise {
			rootPub = contractArgs[5]
			path1 = contractArgs[6]
			path2 = contractArgs[7]

			tpl, errArgs = addPubValueWitness(tpl, rootPub, path1, path2, selector)
		} else if selector == ClauseExpire {
			tpl, errArgs = addValueWitness(tpl, selector)
		}

	default:
		buf := fmt.Sprintf("Invalid contract template name")
		err = errors.New(buf)
		return nil, err
	}

	if errArgs != nil {
		return nil, errArgs
	}

	return tpl, nil
}
