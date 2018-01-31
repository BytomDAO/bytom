package contract

import (
	"fmt"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
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
func BuildContractTransaction(contractName string, args []string, minArgsCount int, alias bool, btmGas string) (*string, error) {
	var buildReqStr *string
	var buf string
	var err error

	if len(args) < minArgsCount {
		buf = fmt.Sprintf("the number of basic arguments[%d] is less than requirements[%d]", len(args), minArgsCount)
		err = errors.New(buf)
		return nil, err
	}

	outputID := args[0]
	accountInfo := args[1]
	assetInfo := args[2]
	amount := args[3]

	switch contractName {
	case "LockWithPublicKey", "LockWithMultiSig", "LockWithPublicKeyHash", "RevealPreimage":
		if alias {
			buf = fmt.Sprintf(buildAcctRecvReqFmtByAlias, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		} else {
			buf = fmt.Sprintf(buildAcctRecvReqFmt, outputID, assetInfo, amount, accountInfo, btmGas, accountInfo)
		}
		buildReqStr = &buf

	case "TradeOffer":
		if buildReqStr, err = buildTradeOfferReq(args, minArgsCount, alias, btmGas); err != nil {
			return nil, err
		}

	case "Escrow":
		if buildReqStr, err = buildEscrowReq(args, minArgsCount, alias, btmGas); err != nil {
			return nil, err
		}

	case "LoanCollateral":
		if buildReqStr, err = buildLoanCollateralReq(args, minArgsCount, alias, btmGas); err != nil {
			return nil, err
		}

	case "CallOption":
		if buildReqStr, err = buildCallOptionReq(args, minArgsCount, alias, btmGas); err != nil {
			return nil, err
		}

	default:
		buf = fmt.Sprintf("Invalid contract template name")
		err = errors.New(buf)
		return nil, err
	}

	return buildReqStr, nil
}

// AddContractArguments add the template contract arguments into txbuilder.Template
func AddContractArguments(tpl *txbuilder.Template, contractName string, contractArgs []string) (*txbuilder.Template, error) {
	var buf string
	var err error

	//add paraments for Witness
	switch contractName {
	case "LockWithPublicKey":
		if len(contractArgs) != 3 {
			buf = fmt.Sprintf("the arguments of contract 'LockWithPublicKey' is not right, Please follow the prompts to add parameters!")
			err = errors.New(buf)
			return nil, err
		}

		pubInfo := newPubKeyInfo(contractArgs[0], []string{contractArgs[1], contractArgs[2]})

		if tpl, err = addPubKeyArgs(tpl, []PubKeyInfo{pubInfo}); err != nil {
			return nil, err
		}

	case "LockWithMultiSig":
		if len(contractArgs) != 6 {
			buf = fmt.Sprintf("the arguments of contract 'LockWithMultiSig' is not right, Please follow the prompts to add parameters!")
			err = errors.New(buf)
			return nil, err
		}

		pubInfo1 := newPubKeyInfo(contractArgs[0], []string{contractArgs[1], contractArgs[2]})
		pubInfo2 := newPubKeyInfo(contractArgs[3], []string{contractArgs[4], contractArgs[5]})

		if tpl, err = addPubKeyArgs(tpl, []PubKeyInfo{pubInfo1, pubInfo2}); err != nil {
			return nil, err
		}

	case "LockWithPublicKeyHash":
		if len(contractArgs) != 4 {
			buf = fmt.Sprintf("the arguments of contract 'LockWithPublicKeyHash' is not right, Please follow the prompts to add parameters!")
			err = errors.New(buf)
			return nil, err
		}

		pubInfo := newPubKeyInfo(contractArgs[1], []string{contractArgs[2], contractArgs[3]})
		paramInfo := newParamInfo([]string{contractArgs[0]}, []PubKeyInfo{pubInfo}, nil)

		if tpl, err = addParamArgs(tpl, paramInfo); err != nil {
			return nil, err
		}

	case "RevealPreimage":
		if len(contractArgs) != 1 {
			buf = fmt.Sprintf("the arguments of contract 'RevealPreimage' is not right, Please follow the prompts to add parameters!")
			err = errors.New(buf)
			return nil, err
		}

		if tpl, err = addDataArgs(tpl, []string{contractArgs[0]}); err != nil {
			return nil, err
		}

	case "TradeOffer":
		if tpl, err = addTradeOfferArgs(tpl, contractArgs); err != nil {
			return nil, err
		}

	case "Escrow":
		if tpl, err = addEscrowArgs(tpl, contractArgs); err != nil {
			return nil, err
		}

	case "LoanCollateral":
		if tpl, err = addLoanCollateralArgs(tpl, contractArgs); err != nil {
			return nil, err
		}

	case "CallOption":
		if tpl, err = addCallOptionArgs(tpl, contractArgs); err != nil {
			return nil, err
		}

	default:
		buf := fmt.Sprintf("Invalid contract template name")
		err = errors.New(buf)
		return nil, err
	}

	return tpl, nil
}
