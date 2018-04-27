package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bytom/api"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/util"
)

const (
	BuildMulTx = "build_mul_tx"
	SignTx     = "Sign_tx"
	SubmitTx   = "submit_tx"
)

var actions = `{"actions": [%s]}`
var feesFmt = `{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount":%s, "account_id": "%s"}`
var inputFmt = `{"type": "spend_account", "asset_id": "%s","amount": %s,"account_id": "%s"}`
var issueInputFmt = `{"type": "issue", "asset_id": "%s", "amount": %s}`
var outputFmt = `{"type": "control_address", "asset_id": "%s", "amount": %s,"address": "%s"}`

var (
	buildType = ""
	btmGas    = "20000000"
	passwd    = "123456"
	baseNum   = 100000000
	index     = 0
)

// SendReq genetate tx and send data
func SendReq(method string, args []string, recvAccount []accountInfo) (interface{}, bool) {
	var param interface{}
	var methodPath string
	switch method {
	case BuildMulTx:
		// send account
		accountInfo := args[0]
		// send btm asset
		assetInfo := args[1]
		bmtTotalAmount := 0
		var (
			input  string
			fees   string
			output string
		)
		// generate output data
		for i := 0; i < len(recvAccount); i++ {
			address := recvAccount[i].address
			bmtTotalAmount += recvAccount[i].amount
			amountTmp := strconv.Itoa(recvAccount[i].amount)
			output += fmt.Sprintf(outputFmt, assetInfo, amountTmp, address)
			output += ","
		}
		amountTmp := strconv.Itoa(bmtTotalAmount)
		btmGasTmp := cfg.BtmGas * 100000000
		btmGas = strconv.Itoa(int(btmGasTmp))
		fees += fmt.Sprintf(feesFmt, btmGas, accountInfo) + ","
		input += fmt.Sprintf(inputFmt, assetInfo, amountTmp, accountInfo)

		buildReqStr := fmt.Sprintf(actions, fees+output+input)
		var ins api.BuildRequest
		if err := json.Unmarshal([]byte(buildReqStr), &ins); err != nil {
			fmt.Println("generate build mul tx is error: ", err)
			os.Exit(util.ErrLocalExe)
		}
		fileName := "build_tx_" + strconv.Itoa(index) + ".txt"
		outputFile, outputError := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if outputError != nil {
			fmt.Println("Failed to open file:", fileName, ",Please check the file.If it exists, please backup.")
			return "", false
		}
		defer outputFile.Close()
		outputWriter := bufio.NewWriter(outputFile)
		rawData, err := json.MarshalIndent(&ins, "", "  ")
		if err != nil {
			fmt.Println("Json format error!!!!!")
			os.Exit(1)
		}
		outputWriter.WriteString(string(rawData))
		outputWriter.Flush()
		fmt.Println(string(rawData))
		fmt.Println("The total number of btm:", bmtTotalAmount/baseNum)
		fmt.Println("Please check the above data or file:[", fileName, "] data")
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
	default:
		fmt.Println("method is null")
		os.Exit(1)
	}
	data, exitCode := util.ClientCall(methodPath, &param)
	if exitCode != util.Success {
		return "", false
	}
	return data, true
}

// Sendbulktx send asset tx
func Sendtx(sendAcct string, sendasset string, recvAccount []accountInfo) {
	//build tx
	var (
		resp interface{}
		b    bool
	)
	param := []string{sendAcct, sendasset}
	resp, b = SendReq(BuildMulTx, param, recvAccount)
	if !b {
		fmt.Println("BuildMulTx fail!")
		os.Exit(1)
	}
	rawTemplate, _ := json.Marshal(resp)
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter yes or no after checking:")
	data, _, _ := reader.ReadLine()
	command := strings.ToLower(strings.TrimSpace(string(data)))
	if command == "yes" {
		tmp := make([]accountInfo, 0)
		//sign
		param = []string{string(rawTemplate)}
		resp, b = SendReq(SignTx, param, tmp)
		if !b {
			fmt.Println("SignTx fail!")
			os.Exit(1)
		}
		// submit
		rawTemplate, _ = json.Marshal(resp)
		var data signResp
		json.Unmarshal(rawTemplate, &data)
		rawTemplate, _ = json.Marshal(*data.Tx)
		param = []string{string(rawTemplate)}
		resp, b = SendReq(SubmitTx, param, tmp)
		if !b {
			fmt.Println("SubmitTx fail!")
			os.Exit(1)
		}
		index += 1
		fmt.Println(resp)
	} else {
		fmt.Println("exit...")
		os.Exit(1)
	}
}
