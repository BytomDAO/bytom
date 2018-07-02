package commands

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
