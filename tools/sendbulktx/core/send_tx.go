package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/bytom/api"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/util"
)

const (
	CreateKey      = "create_key"
	ListKeys       = "list_keys"
	CreateAccount  = "create_account"
	CreateAsset    = "create_asset"
	CreateReceiver = "CreateReceiver"
	BuildTx        = "build_tx"
	BuildMulTx     = "build_mul_tx"
	BuildSpend     = "spend"
	BuildIssue     = "issue"
	BuildCtlAddr   = "address"
	SignTx         = "Sign_tx"
	SubmitTx       = "submit_tx"
	GetTransaction = "get_transaction"
)

var buildIssueReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "issue", "asset_id": "%s", "amount": %s},
		{"type": "control_address", "asset_id": "%s", "amount": %s, "address": "%s"}
	]}`

var buildSpendReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_receiver", "asset_id": "%s", "amount": %s, "receiver":{"control_program": "%s"}}
	]}`

var buildControlAddressReqFmt = `
	{"actions": [
		{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"},
		{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"},
		{"type": "control_address", "asset_id": "%s", "amount": %s,"address": "%s"}
	]}`

var actions = `{"actions": [%s]}`
var feesFmt = `{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}`
var inputFmt = `{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"}`
var issueInputFmt = `{"type": "issue", "asset_id": "%s", "amount": %s}`
var outputFmt = `{"type": "control_address", "asset_id": "%s", "amount": %s,"address": "%s"}`

var (
	buildType     = ""
	btmGas        = "200000000"
	accountQuorum = 1
	passwd        = "123456"
)

// RestoreStruct Restore data
func RestoreStruct(data interface{}, out interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if ok != true {
		fmt.Println("invalid type assertion")
		os.Exit(util.ErrLocalParse)
	}

	rawData, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(util.ErrLocalParse)
	}
	json.Unmarshal(rawData, out)
}

// SendReq genetate tx and send data
func SendReq(method string, args []string) (interface{}, bool) {
	var param interface{}
	var methodPath string
	switch method {
	case CreateKey:
		ins := keyIns{Alias: args[0], Password: args[1]}
		param = ins
		methodPath = "/create-key"
	case ListKeys:
		methodPath = "/list-keys"
	case CreateAccount:
		ins := account{}
		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			fmt.Println("CreateAccount error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		ins.RootXPubs = append(ins.RootXPubs, xpub)
		ins.Quorum = accountQuorum
		ins.Alias = args[0]
		ins.AccessToken = ""
		param = ins
		methodPath = "/create-account"
	case CreateAsset:
		ins := asset{}
		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(args[1])); err != nil {
			fmt.Println("CreateAsset error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		ins.RootXPubs = append(ins.RootXPubs, xpub)
		ins.Quorum = 1
		ins.Alias = args[0]
		ins.AccessToken = ""
		param = ins
		methodPath = "/create-asset"
	case CreateReceiver:
		var ins = Reveive{AccountAlias: args[0]}
		param = ins
		methodPath = "/create-account-receiver"
	case BuildTx:
		accountInfo := args[0]
		assetInfo := args[1]
		amount := args[2]
		receiverProgram := args[3]
		buildType := args[4]
		var buildReqStr string

		switch buildType {
		case BuildSpend:
			buildReqStr = fmt.Sprintf(buildSpendReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, receiverProgram)
		case BuildIssue:
			address := args[3]
			buildReqStr = fmt.Sprintf(buildIssueReqFmt, btmGas, accountInfo, assetInfo, amount, assetInfo, amount, address)
		case BuildCtlAddr:
			address := args[3]
			buildReqStr = fmt.Sprintf(buildControlAddressReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, address)
		default:
			buildReqStr = fmt.Sprintf(buildSpendReqFmt, btmGas, accountInfo, assetInfo, amount, accountInfo, assetInfo, amount, receiverProgram)

		}
		var ins api.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &ins); err != nil {
			fmt.Println("generate build tx is error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		param = ins
		methodPath = "/build-transaction"
	case BuildMulTx:
		accountInfo := args[0]
		assetInfo := args[1]
		amount := args[2]
		receiverProgram := args[3]
		buildType := args[4]
		var (
			input  string
			fees   string
			output string
		)
		fees += fmt.Sprintf(feesFmt, btmGas, accountInfo) + ","
		switch buildType {
		case BuildIssue:
			amountInt, _ := strconv.Atoi(amount)
			amountTmp := strconv.Itoa(amountInt * mulOutput)
			input += fmt.Sprintf(issueInputFmt, assetInfo, amountTmp)
			input += ","
		case BuildCtlAddr:
			amountInt, _ := strconv.Atoi(amount)
			amountTmp := strconv.Itoa(amountInt * mulOutput)
			input += fmt.Sprintf(inputFmt, assetInfo, amountTmp, accountInfo)
			input += ","
		default:
			fmt.Println("buildType:[", buildType, "] do not implemented")
			os.Exit(1)
		}
		for i := 0; i < mulOutput; i++ {
			output += fmt.Sprintf(outputFmt, assetInfo, amount, receiverProgram)
			if i < mulOutput-1 {
				output += ","
			}
		}
		buildReqStr := fmt.Sprintf(actions, fees+input+output)
		var ins api.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &ins); err != nil {
			fmt.Println("generate build mul tx is error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		param = ins
		methodPath = "/build-transaction"

	case SignTx:
		template := txbuilder.Template{}

		err := json.Unmarshal([]byte(args[0]), &template)
		if err != nil {
			fmt.Println(err)
			os.Exit(util.ErrLocalExe)
		}

		var ins = struct {
			Password string             `json:"password"`
			Txs      txbuilder.Template `json:"transaction"`
		}{Password: cfg.Password, Txs: template}

		param = ins
		methodPath = "/sign-transaction"
	case SubmitTx:
		var ins = struct {
			Tx types.Tx `json:"raw_transaction"`
		}{}
		json.Unmarshal([]byte(args[0]), &ins)
		methodPath = "/submit-transaction"
		data, exitCode := util.ClientCall(methodPath, &ins)
		if exitCode != util.Success {
			return "", false
		}
		return data, true
	case GetTransaction:
		ins := &struct {
			TxID string `json:"tx_id"`
		}{TxID: args[0]}
		param = ins
		methodPath = "/get-transaction"
	default:
		return "", false
	}
	data, exitCode := ClientCall(methodPath, &param)
	if exitCode != util.Success {
		return "", false
	}
	return data, true
}

// Sendbulktx send asset tx
func Sendbulktx(threadTxNum int, txBtmNum string, sendAcct string, sendasset string, controlPrograms []string, txidChan chan string, index *uint64) {
	arrayLen := len(controlPrograms)
	for i := 0; i < threadTxNum; i++ {
		//build tx
		receiver := controlPrograms[i%arrayLen]
		if strings.EqualFold(receiver, "") {
			txidChan <- ""
			continue
		}
		var (
			resp interface{}
			b    bool
		)
		param := []string{sendAcct, sendasset, txBtmNum, receiver, cfg.BuildType}
		if mulOutput > 0 {
			resp, b = SendReq(BuildMulTx, param)
		} else {
			resp, b = SendReq(BuildTx, param)
		}
		if !b {
			txidChan <- ""
			continue
		}
		rawTemplate, _ := json.Marshal(resp)
		//sign
		param = []string{string(rawTemplate)}
		resp, b = SendReq(SignTx, param)
		if !b {
			fmt.Println("SignTx fail")
			txidChan <- ""
			continue
		}
		// submit
		rawTemplate, _ = json.Marshal(resp)
		var data signResp
		json.Unmarshal(rawTemplate, &data)
		rawTemplate, _ = json.Marshal(*data.Tx)
		param = []string{string(rawTemplate)}
		resp, b = SendReq(SubmitTx, param)
		if !b {
			fmt.Println("SubmitTx fail")
			txidChan <- ""
			continue
		}

		atomic.AddUint64(index, 1)
		fmt.Println("tx num:", atomic.LoadUint64(index), " txid:", resp)
		type txID struct {
			Txid string `json:"tx_id"`
		}
		var out txID
		RestoreStruct(resp, &out)
		txidChan <- out.Txid
	}
}
