package contract

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
