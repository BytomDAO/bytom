package commands

import (
	"fmt"

	"github.com/bytom/errors"
)

const (
	// contract TradeOffer's clause
	clauseTrade      = "00000000"
	clauseCancel     = "13000000"
	tradeOfferEnding = "1a000000"

	// contract Escrow's clause
	clauseApprove = "00000000"
	clauseReject  = "1b000000"
	escrowEnding  = "2a000000"
)

var (
	errBadContractArguments = errors.New("bad contract arguments")
)

// contract is LockWithPublicKey
var buildLockWithPublicKeyReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %s, "account_id": "%s"}
	]}`

var buildLockWithPublicKeyReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"}
	]}`

// contract is LockWithMultiSig
var buildLockWithMultiSigReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}},
				{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %s, "account_id": "%s"}
	]}`

var buildLockWithMultiSigReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}},
				{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"}
	]}`

// contract is LockWithPublicKeyHash
var buildLockWithPublicKeyHashReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "data", "raw_data": {"value": "%s"}},
				{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %s, "account_id": "%s"}
	]}`

var buildLockWithPublicKeyHashReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "data", "raw_data": {"value": "%s"}},
				{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"}
	]}`

// contract is RevealPreimage
var buildRevealPreimageReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %s, "account_id": "%s"}
	]}`

var buildRevealPreimageReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"}
	]}`

// contract is TradeOffer's clause trade, the code of clause contains only two statement with "lock payment with program" and "unlock value"
var buildTradeOfferClauseTradeReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s", "arguments": [{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "%s", "amount": %s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"}
	]}`

var buildTradeOfferClauseTradeReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s", "arguments": [{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "%s", "amount": %s, "account_alias": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"}
	]}`

// contract is TradeOffer's clause cancel
var buildTradeOfferClauseCancelReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}},
				{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %s, "account_id": "%s"}
	]}`

var buildTradeOfferClauseCancelReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}},
				{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"}
	]}`

// contract is Escrow
var buildEscrowReqFmt = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id": "%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}},
				{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_id": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}
	]}`

var buildEscrowReqFmtByAlias = `
	{"actions": [
		{"type": "spend_account_unspent_output", "output_id":"%s", "arguments": [{"type": "raw_tx_signature", "raw_data": {"xpub": "%s", "derivation_path": ["%s", "%s"]}},
				{"type": "data", "raw_data": {"value": "%s"}}]},
		{"type": "control_program", "asset_alias": "%s", "amount": %s, "control_program": "%s"},
		{"type": "spend_account", "asset_alias": "BTM", "amount": %s, "account_alias": "%s"}
	]}`

// contract arguments
type baseContractArg struct {
	accountInfo string
	assetInfo   string
	amount      string
	alias       bool
	program     string
	btmGas      string
	outputID    string
}

type basePubInfo struct {
	rootPub string
	path1   string
	path2   string
}

type innerContractArg struct {
	innerAccountInfo string
	innerAssetInfo   string
	innerAmount      string
	innerProgram     string
}

// addContractArgs add arguments for template contracts
func addContractArgs(contractName string, baseArg baseContractArg, specArgs []string, usage string) (buildReqStr string, err error) {
	switch contractName {
	case "LockWithPublicKey":
		if len(specArgs) != 3 {
			err = errors.WithDetailf(errBadContractArguments, "%s <rootPub> <path1> <path2> [flags]\n", usage)
			return
		}

		pubInfo := basePubInfo{
			rootPub: specArgs[0],
			path1:   specArgs[1],
			path2:   specArgs[2],
		}
		buildReqStr = addLockWithPublicKeyArg(baseArg, pubInfo)

	case "LockWithMultiSig":
		if len(specArgs) != 6 {
			err = errors.WithDetailf(errBadContractArguments, "%s <rootPub1> <path11> <path12> <rootPub2> <path21> <path22> [flags]\n", usage)
			return
		}

		pubInfos := [2]basePubInfo{
			{
				rootPub: specArgs[0],
				path1:   specArgs[1],
				path2:   specArgs[2],
			},
			{
				rootPub: specArgs[3],
				path1:   specArgs[4],
				path2:   specArgs[5],
			},
		}
		buildReqStr = addLockWithMultiSigArg(baseArg, pubInfos)

	case "LockWithPublicKeyHash":
		if len(specArgs) != 4 {
			err = errors.WithDetailf(errBadContractArguments, "%s <pubKey> <rootPub> <path1> <path2> [flags]\n", usage)
			return
		}

		pubkey := specArgs[0]
		pubInfo := basePubInfo{
			rootPub: specArgs[1],
			path1:   specArgs[2],
			path2:   specArgs[3],
		}
		buildReqStr = addLockWithPublicKeyHashArg(baseArg, pubInfo, pubkey)

	case "RevealPreimage":
		if len(specArgs) != 1 {
			err = errors.WithDetailf(errBadContractArguments, "%s <value> [flags]\n", usage)
			return
		}

		value := specArgs[0]
		buildReqStr = addRevealPreimageArg(baseArg, value)

	case "TradeOffer":
		switch {
		case len(specArgs) <= 0:
			err = errors.WithDetailf(errBadContractArguments, "%s <clauseSelector> (<innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram>) | (<rootPub> <path1> <path2>) [flags]\n", usage)
		case specArgs[0] == clauseTrade:
			if len(specArgs) != 5 {
				err = errors.WithDetailf(errBadContractArguments, "%s <clauseSelector> <innerAccountID|alias> <innerAssetID|alias> <innerAmount> <innerProgram> [flags]\n", usage)
				return
			}

			inner := &innerContractArg{
				innerAccountInfo: specArgs[1],
				innerAssetInfo:   specArgs[2],
				innerAmount:      specArgs[3],
				innerProgram:     specArgs[4],
			}
			buildReqStr, err = addTradeOfferArg(baseArg, clauseTrade, inner, nil)

		case specArgs[0] == clauseCancel:
			if len(specArgs) != 4 {
				err = errors.WithDetailf(errBadContractArguments, "%s <clauseSelector> <rootPub> <path1> <path2> [flags]\n", usage)
				return
			}

			pubInfo := &basePubInfo{
				rootPub: specArgs[1],
				path1:   specArgs[2],
				path2:   specArgs[3],
			}
			buildReqStr, err = addTradeOfferArg(baseArg, clauseCancel, nil, pubInfo)

		case specArgs[0] == tradeOfferEnding:
			err = errors.WithDetailf(errBadContractArguments, "Clause ending was selected in contract %s, ending exit\n", contractName)
		default:
			err = errors.WithDetailf(errBadContractArguments, "selected clause [%s] error, contract %s's clause must in set [%s, %s, %s]\n",
				specArgs[0], contractName, clauseTrade, clauseCancel, tradeOfferEnding)
		}

	case "Escrow":
		switch {
		case len(specArgs) <= 0:
			err = errors.WithDetailf(errBadContractArguments, "%s <clauseSelector> <rootPub> <path1> <path2> <controlProgram> [flags]\n", usage)
		case specArgs[0] == clauseApprove || specArgs[0] == clauseReject:
			if len(specArgs) != 5 {
				err = errors.WithDetailf(errBadContractArguments, "%s <clauseSelector> <rootPub> <path1> <path2> <controlProgram> [flags]\n", usage)
				return
			}

			pubInfo := &basePubInfo{
				rootPub: specArgs[1],
				path1:   specArgs[2],
				path2:   specArgs[3],
			}
			controlProgram := specArgs[4]
			buildReqStr, err = addEscrowArg(baseArg, specArgs[0], pubInfo, controlProgram)

		case specArgs[0] == escrowEnding:
			err = errors.WithDetailf(errBadContractArguments, "Clause ending was selected in contract %s, ending exit\n", contractName)
		default:
			err = errors.WithDetailf(errBadContractArguments, "selected clause [%s] error, contract %s's clause must in set [%s, %s, %s]\n",
				specArgs[0], contractName, clauseApprove, clauseReject, escrowEnding)
		}

	default:
		err = errors.WithDetailf(errBadContractArguments, "Invalid contract template name [%s]", contractName)
	}

	return
}

func addLockWithPublicKeyArg(baseArg baseContractArg, pubInfo basePubInfo) (buildReqStr string) {
	buildReqStr = fmt.Sprintf(buildLockWithPublicKeyReqFmt, baseArg.outputID, pubInfo.rootPub, pubInfo.path1, pubInfo.path2,
		baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	if baseArg.alias {
		buildReqStr = fmt.Sprintf(buildLockWithPublicKeyReqFmtByAlias, baseArg.outputID, pubInfo.rootPub, pubInfo.path1, pubInfo.path2,
			baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	}

	return
}

func addLockWithMultiSigArg(baseArg baseContractArg, pubInfos [2]basePubInfo) (buildReqStr string) {
	buildReqStr = fmt.Sprintf(buildLockWithMultiSigReqFmt, baseArg.outputID, pubInfos[0].rootPub, pubInfos[0].path1, pubInfos[0].path2,
		pubInfos[1].rootPub, pubInfos[1].path1, pubInfos[1].path2,
		baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	if baseArg.alias {
		buildReqStr = fmt.Sprintf(buildLockWithMultiSigReqFmtByAlias, baseArg.outputID, pubInfos[0].rootPub, pubInfos[0].path1, pubInfos[0].path2,
			pubInfos[1].rootPub, pubInfos[1].path1, pubInfos[1].path2,
			baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	}

	return
}

func addLockWithPublicKeyHashArg(baseArg baseContractArg, pubInfo basePubInfo, pubkey string) (buildReqStr string) {
	buildReqStr = fmt.Sprintf(buildLockWithPublicKeyHashReqFmt, baseArg.outputID, pubkey, pubInfo.rootPub, pubInfo.path1, pubInfo.path2,
		baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	if alias {
		buildReqStr = fmt.Sprintf(buildLockWithPublicKeyHashReqFmtByAlias, baseArg.outputID, pubkey, pubInfo.rootPub, pubInfo.path1, pubInfo.path2,
			baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	}

	return
}

func addRevealPreimageArg(baseArg baseContractArg, value string) (buildReqStr string) {
	buildReqStr = fmt.Sprintf(buildRevealPreimageReqFmt, baseArg.outputID, value,
		baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	if baseArg.alias {
		buildReqStr = fmt.Sprintf(buildRevealPreimageReqFmtByAlias, baseArg.outputID, value,
			baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
	}

	return
}

func addTradeOfferArg(baseArg baseContractArg, selector string, innerArg *innerContractArg, pubInfo *basePubInfo) (buildReqStr string, err error) {
	switch selector {
	case clauseTrade:
		if innerArg == nil {
			err = errors.New("Contract TradeOffer's clause trade argument is nil")
			return
		}

		buildReqStr = fmt.Sprintf(buildTradeOfferClauseTradeReqFmt, baseArg.outputID, clauseTrade,
			innerArg.innerAssetInfo, innerArg.innerAmount, innerArg.innerProgram,
			innerArg.innerAssetInfo, innerArg.innerAmount, innerArg.innerAccountInfo,
			baseArg.btmGas, baseArg.accountInfo,
			baseArg.assetInfo, baseArg.amount, baseArg.program)
		if baseArg.alias {
			buildReqStr = fmt.Sprintf(buildTradeOfferClauseTradeReqFmtByAlias, baseArg.outputID, clauseTrade,
				innerArg.innerAssetInfo, innerArg.innerAmount, innerArg.innerProgram,
				innerArg.innerAssetInfo, innerArg.innerAmount, innerArg.innerAccountInfo,
				baseArg.btmGas, baseArg.accountInfo,
				baseArg.assetInfo, baseArg.amount, baseArg.program)
		}

	case clauseCancel:
		if pubInfo == nil {
			err = errors.New("Contract TradeOffer's clause cancel argument is nil")
			return
		}

		buildReqStr = fmt.Sprintf(buildTradeOfferClauseCancelReqFmt, baseArg.outputID, pubInfo.rootPub, pubInfo.path1, pubInfo.path2, clauseCancel,
			baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
		if baseArg.alias {
			buildReqStr = fmt.Sprintf(buildTradeOfferClauseCancelReqFmtByAlias, baseArg.outputID, pubInfo.rootPub, pubInfo.path1, pubInfo.path2, clauseCancel,
				baseArg.assetInfo, baseArg.amount, baseArg.program, baseArg.btmGas, baseArg.accountInfo)
		}

	default:
		err = errors.New("Invalid contract clause selector")
	}

	return
}

func addEscrowArg(baseArg baseContractArg, selector string, pubInfo *basePubInfo, controlProgram string) (buildReqStr string, err error) {
	switch selector {
	case clauseApprove, clauseReject:
		if pubInfo == nil {
			err = errors.New("Contract Escrow's clause argument is nil")
			return
		}

		buildReqStr = fmt.Sprintf(buildEscrowReqFmt, baseArg.outputID, pubInfo.rootPub, pubInfo.path1, pubInfo.path2, selector,
			baseArg.assetInfo, baseArg.amount, controlProgram, baseArg.btmGas, baseArg.accountInfo)
		if baseArg.alias {
			buildReqStr = fmt.Sprintf(buildEscrowReqFmtByAlias, baseArg.outputID, pubInfo.rootPub, pubInfo.path1, pubInfo.path2, selector,
				baseArg.assetInfo, baseArg.amount, controlProgram, baseArg.btmGas, baseArg.accountInfo)
		}

	default:
		err = errors.New("Invalid contract clause selector")
	}

	return
}
