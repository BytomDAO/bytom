package example

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/encoding/json"
	"github.com/bytom/config"

	stdjson "encoding/json"
	bchain "github.com/bytom/blockchain"
)

const (
	Account = "account"
	Asset   = "asset"
)

type Ins struct {
	RootXPubs   []chainkd.XPub `json:"root_xpubs"`
	Quorum      int
	Alias       string
	Tags        map[string]interface{}
	Definition  map[string]interface{} `json:"omitempty"`
	ClientToken string                 `json:"client_token"`
}

func NewInstance(alias, typ string) (Ins, chainkd.XPrv) {
	xprv, _ := chainkd.NewXPrv(nil)
	xpub := xprv.XPub()
	fmt.Printf("type:%s,xprv:%v\n", typ, xprv)
	fmt.Printf("type:%s,xpub:%v\n", typ, xpub)

	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = alias
	ins.Tags = map[string]interface{}{"test_tag": "v0"}
	if typ == Asset {
		ins.Definition = map[string]interface{}{"test_definition": "v0"}
	}
	ins.ClientToken = typ

	return ins, xprv
}

func NewAnnotate(client *rpc.Client, typ string, ins ...Ins) ([]query.AnnotatedAccount, []query.AnnotatedAsset) {
	accounts := make([]query.AnnotatedAccount, 1)
	assets := make([]query.AnnotatedAsset, 1)

	if typ == Account {
		client.Call(context.Background(), "/create-account", &ins, &accounts)
		fmt.Printf("account:%v\n", accounts)
		return accounts, nil
	} else if typ == Asset {
		client.Call(context.Background(), "/create-asset", &ins, &assets)
		fmt.Printf("assetid=%s\n", assets[0].ID.String())
		fmt.Printf("asset:%v\n", assets)
		return nil, assets
	}

	return nil, nil
}

func IssueTest(client *rpc.Client, args []string) {
	// Create Account.
	fmt.Printf("To create Account:\n")
	aliceIns, _ := NewInstance("alice", Account)
	bobIns, _ := NewInstance("bob", Account)
	accounts, _ := NewAnnotate(client, Account, aliceIns, bobIns)

	// Create Asset.
	fmt.Printf("To create Asset:\n")
	goldIns, xprvGold := NewInstance("gold", Asset)
	_, assets := NewAnnotate(client, Asset, goldIns)

	// Build Transaction.
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
	fmt.Printf("spend_account_unspent_output:%v\n", config.GenerateGenesisTx().ResultIds[0])
	buildReqStr := fmt.Sprintf(buildReqFmt, config.GenerateGenesisTx().ResultIds[0], assets[0].ID.String(), assets[0].ID.String(), accounts[0].ID, accounts[0].ID)
	var buildReq bchain.BuildRequest
	err := stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		fmt.Println(err)
	}

	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*bchain.BuildRequest{&buildReq}, &tpl)
	fmt.Printf("tpl:%v\n", tpl)

	// sign-transaction
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

	// submit-transaction
	var submitResponse interface{}
	submitArg := bchain.SubmitArg{Transactions: tpl, Wait: json.Duration{Duration: time.Duration(1000000)}, WaitUntil: "none"}
	client.Call(context.Background(), "/submit-transaction", submitArg, &submitResponse)
	fmt.Printf("submit transaction:%v\n", submitResponse)
	fmt.Println("==============test end===============")

	//Issue result:
	//alice <btm:8888888888,gold:100>
}
