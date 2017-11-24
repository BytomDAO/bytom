package example

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	bchain "github.com/bytom/blockchain"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/encoding/json"
	"github.com/bytom/config"
)

type accUTXOShort struct {
	OutputID  string `json:"OutputID"`
	AccountID string `json:"AccountID"`
	AssetID   string `json:"AssetID"`
	Amount    string `json:"Amount"`
}

type requestQuery struct {
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
	PageSize     int           `json:"page_size"`
	AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
	Timeout      json.Duration `json:"timeout"`
	After        string        `json:"after"`
	StartTimeMS  uint64        `json:"start_time,omitempty"`
	EndTimeMS    uint64        `json:"end_time,omitempty"`
	TimestampMS  uint64        `json:"timestamp,omitempty"`
	Type         string        `json:"type"`
	Aliases      []string      `json:"aliases,omitempty"`
}

func CoinbaseTest(client *rpc.Client, args []string) {
	if len(args) != 3 {
		fmt.Println("error: CoinbaseTest need accountID")
		os.Exit(1)
	}

	buildReqFmt := `
	{
		"actions": [
            {"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 624000000000, "account_id": "%s"},
            {"type": "control_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 8888888888, "account_id": "%s"}
        ]
    }`
	buildReqStr := fmt.Sprintf(buildReqFmt, args[0], args[0])

	var buildReq bchain.BuildRequest
	if err := stdjson.Unmarshal([]byte(buildReqStr), &buildReq); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*bchain.BuildRequest{&buildReq}, &tpl)

	tx := tpl[0].Transaction
	for i, input := range tx.Inputs {
		assetID := input.AssetID()
		fmt.Printf("Input #%d --- asset: %s, amount: %d\n", i, assetID.String(), input.Amount())
	}
	for i, output := range tx.Outputs {
		fmt.Printf("Output #%d --- asset: %s, amount: %d\n", i, output.AssetId.String(), output.Amount)
	}
}

func SpendTest(client *rpc.Client, args []string) {
	// Create Account.
	fmt.Printf("To create Account:\n")
	aliceIns, xprvAlice := NewInstance("alice", Account)
	bobIns, _ := NewInstance("bob", Account)
	accounts, _ := NewAnnotate(client, Account, aliceIns, bobIns)

	// Create Asset.
	fmt.Printf("To create Asset:\n")
	goldIns, xprvGold := NewInstance("gold", Asset)
	_, assets := NewAnnotate(client, Asset, goldIns)

	// Build Transaction1-Issue
	fmt.Printf("To build transaction:\n")
	buildReqFmt := `
		{"actions": [
			{
				"type":"spend_account_unspent_output",
				"receiver":null,
				"output_id":"%v",
				"reference_data":{}
			},
			{"type": "issue", "asset_id": "%s", "amount": 100},
			{"type": "control_account", "asset_id": "%s", "amount": 100, "account_id": "%s"},
			{"type": "control_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 8888888888, "account_id": "%s"}
		]}`
	buildReqStr := fmt.Sprintf(buildReqFmt, config.GenerateGenesisTx().ResultIds[0], assets[0].ID.String(), assets[0].ID.String(), accounts[0].ID, accounts[0].ID)
	var buildReq bchain.BuildRequest
	err := stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		fmt.Println(err)
	}

	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*bchain.BuildRequest{&buildReq}, &tpl)
	fmt.Printf("tpl:%v\n", tpl)

	// sign-transaction1-Issue
	err = txbuilder.Sign(context.Background(), &tpl[0], []chainkd.XPub{xprvGold.XPub()}, "", func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte, _ string) ([]byte, error) {
		derived := xprvGold.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		fmt.Printf("sign-transaction error. err:%v\n", err)
		os.Exit(0)
	}

	fmt.Printf("sign tpl:%v\n", tpl[0])
	fmt.Printf("sign tpl's SigningInstructions:%v\n", tpl[0].SigningInstructions[0])
	fmt.Printf("SigningInstructions's SignatureWitnesses:%v\n", tpl[0].SigningInstructions[0].SignatureWitnesses[0])

	// submit-transaction1-Issue
	var submitResponse interface{}
	submitArg := bchain.SubmitArg{Transactions: tpl, Wait: json.Duration{Duration: time.Duration(1000000)}, WaitUntil: "none"}
	client.Call(context.Background(), "/submit-transaction", submitArg, &submitResponse)
	fmt.Printf("submit transaction:%v\n", submitResponse)

	//Issue result:
	//alice <btm:8888888888,gold:100>

	fmt.Println("===========================wait to buid accountutxos.db===============================================")
	time.Sleep(time.Second * 8)

	// Build Transaction2-Spend_account
	fmt.Printf("To build transaction2:\n")
	buildReqFmt2 := `
		{"actions": [
			{"type": "spend_account", "asset_id": "%s", "amount": 40, "account_id": "%s"},
			{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 10000000, "account_id": "%s"},
			{"type": "control_account", "asset_id": "%s", "amount": 40, "account_id": "%s"},
			{"type": "control_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 5000000, "account_id": "%s"}
	]}`
	buildReqStr2 := fmt.Sprintf(buildReqFmt2, assets[0].ID.String(), accounts[0].ID, accounts[0].ID, assets[0].ID.String(), accounts[1].ID, accounts[1].ID)

	var buildReq2 bchain.BuildRequest
	err = stdjson.Unmarshal([]byte(buildReqStr2), &buildReq2)
	if err != nil {
		fmt.Println(err)
	}

	tpl2 := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*bchain.BuildRequest{&buildReq2}, &tpl2)
	fmt.Printf("tpl2:%v\n", tpl2)

	// sign-transaction2-Spend_account
	err = txbuilder.Sign(context.Background(), &tpl2[0], []chainkd.XPub{xprvAlice.XPub()}, "", func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte, _ string) ([]byte, error) {
		derived := xprvAlice.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		fmt.Printf("sign-transaction2 error. err:%v\n", err)
		os.Exit(0)
	}

	fmt.Printf("sign tpl2:%v\n", tpl2[0])
	fmt.Printf("sign tpl2's SigningInstructions:%v\n", tpl2[0].SigningInstructions[0])
	fmt.Printf("SigningInstructions's SignatureWitnesses:%v\n", tpl2[0].SigningInstructions[0].SignatureWitnesses[0])

	// submit-transaction2-Spend_account
	var submitResponse2 interface{}
	submitArg2 := bchain.SubmitArg{Transactions: tpl2, Wait: json.Duration{Duration: time.Duration(1000000)}, WaitUntil: "none"}
	client.Call(context.Background(), "/submit-transaction", submitArg2, &submitResponse2)
	fmt.Printf("submit2 transaction:%v\n", submitResponse2)

	//Spend_account result:
	//alice <btm:8888888888-10000000,gold:60>
	//bob <btm:5000000,gold:40>
	//fee 10000000-5000000

	fmt.Println("===========================wait to buid accountutxos.db===============================================")
	time.Sleep(time.Second * 8)

	// Build Transaction3-Spend_account_utxo
	fmt.Printf("To build transaction3:\n")

	// Get one UTXO
	var tmp accUTXOShort
	var in requestQuery
	var amount uint64

	responses := make([]interface{}, 0)

	client.Call(context.Background(), "/list-unspent-outputs", in, &responses)
	if len(responses) > 0 {
		for i, item := range responses {
			itemString, _ := item.(string)
			err = stdjson.Unmarshal(stdjson.RawMessage(itemString), &tmp)
			if err != nil {
				fmt.Printf("Spend_account_utxo: test fail, err:%v\n", err)
				os.Exit(0)
			}
			if accounts[0].ID == tmp.AccountID &&
				assets[0].ID.String() == tmp.AssetID {
				//get one alice gold utxo
				fmt.Println(i, "-----", item)
				break
			}
		}
	}

	if tmp.AccountID == "" {
		fmt.Printf("Spend_account_utxo: get on utxo fail\n")
		os.Exit(0)
	}
	amount, _ = strconv.ParseUint(tmp.Amount, 10, 64)
	fmt.Printf("Get one accUTXOShort: %v\n", tmp)

	buildReqFmt3 := `
		{"actions": [
			{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": 10000000, "account_id": "%s"},
			{"type": "spend_account_unspent_output", "output_id": "%s"},
			{"type": "control_account", "asset_id": "%s", "amount": %d, "account_id": "%s"}
	]}`
	buildReqStr3 := fmt.Sprintf(buildReqFmt3, accounts[0].ID, tmp.OutputID, tmp.AssetID, amount, accounts[1].ID)

	var buildReq3 bchain.BuildRequest
	err = stdjson.Unmarshal([]byte(buildReqStr3), &buildReq3)
	if err != nil {
		fmt.Println(err)
	}

	tpl3 := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*bchain.BuildRequest{&buildReq3}, &tpl3)
	fmt.Printf("tpl3:%v\n", tpl3)

	// sign-transaction3-Spend_account_utxo
	err = txbuilder.Sign(context.Background(), &tpl3[0], []chainkd.XPub{xprvAlice.XPub()}, "", func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte, _ string) ([]byte, error) {
		derived := xprvAlice.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		fmt.Printf("sign-transaction3 error. err:%v\n", err)
		os.Exit(0)
	}

	fmt.Printf("sign tpl3:%v\n", tpl2[0])
	fmt.Printf("sign tpl3's SigningInstructions:%v\n", tpl3[0].SigningInstructions[0])
	fmt.Printf("SigningInstructions's SignatureWitnesses:%v\n", tpl3[0].SigningInstructions[0].SignatureWitnesses[0])

	// submit-transaction3-Spend_account_utxo
	var submitResponse3 interface{}
	submitArg3 := bchain.SubmitArg{Transactions: tpl3, Wait: json.Duration{Duration: time.Duration(1000000)}, WaitUntil: "none"}
	client.Call(context.Background(), "/submit-transaction", submitArg3, &submitResponse3)
	fmt.Printf("submit3 transaction:%v\n", submitResponse3)
	fmt.Println("==============test end===============")
	//Spend_account_utxo result:
	//alice <btm:8888888888-10000000-10000000,gold:0>
	//bob <btm:5000000,gold:100>
	//fee 10000000
}
