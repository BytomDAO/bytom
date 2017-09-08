package example

import (
	//"bytes"
	"context"
//	"flag"
	"fmt"
	//"io"
	//"net"
	//"net/http"
	//"os"
	//"path/filepath"
	//"strings"
	//"time"
	stdjson "encoding/json"

	//"github.com/bytom/blockchain"
	"github.com/bytom/blockchain/rpc"
	//"github.com/bytom/crypto/ed25519"
	//"github.com/bytom/env"
	//"github.com/bytom/errors"
	//"github.com/bytom/log"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/txbuilder"
	bc "github.com/bytom/blockchain"
)

// TO DO: issue a asset to a account.
func IssueTest(client *rpc.Client, args []string) {
	// Create Account.
	fmt.Printf("To create Account:\n")
	xprv, _ := chainkd.NewXPrv(nil)
	xpub := xprv.XPub()
	fmt.Printf("xprv_account:%v\n", xprv)
	fmt.Printf("xpub_account:%v\n", xpub)
	type Ins struct {
	    RootXPubs []chainkd.XPub `json:"root_xpubs"`
		Quorum    int
		Alias     string
		Tags      map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins Ins
	ins.RootXPubs = []chainkd.XPub{xpub}
	ins.Quorum = 1
	ins.Alias = "alice"
	ins.Tags = map[string]interface{}{"test_tag": "v0",}
	ins.ClientToken = "account"
	account := make([]query.AnnotatedAccount, 1)
	client.Call(context.Background(), "/create-account", &[]Ins{ins,}, &account)
	fmt.Printf("account:%v\n", account)


	// Create Asset.
	fmt.Printf("To create Asset:\n")
	xprv_asset, _ := chainkd.NewXPrv(nil)
	xpub_asset := xprv_asset.XPub()
	fmt.Printf("xprv_asset:%v\n", xprv_asset)
	fmt.Printf("xpub_asset:%v\n", xpub_asset)
	type Ins_asset struct {
	    RootXPubs []chainkd.XPub `json:"root_xpubs"`
		Quorum    int
		Alias     string
		Tags      map[string]interface{}
		Definition  map[string]interface{}
		ClientToken string `json:"client_token"`
	}
	var ins_asset Ins_asset
	ins_asset.RootXPubs = []chainkd.XPub{xpub_asset}
	ins_asset.Quorum = 1
	ins_asset.Alias = "gold"
	ins_asset.Tags = map[string]interface{}{"test_tag": "v0",}
	ins_asset.Definition = map[string]interface{}{"test_definition": "v0"}
	ins_asset.ClientToken = "asset"
	asset := make([]query.AnnotatedAsset, 1)
	client.Call(context.Background(), "/create-asset", &[]Ins_asset{ins_asset,}, &asset)
	fmt.Printf("asset:%v\n", asset)


	// Build Transaction.
	fmt.Printf("To build transaction:\n")
	// Now Issue actions
	buildReqFmt := `
		{"actions": [
			{"type": "issue", "asset_id": "%s", "amount": 100},
			{"type": "control_account", "asset_id": "%s", "amount": 100, "account_id": "%s"}
		]}`
	buildReqStr := fmt.Sprintf(buildReqFmt, asset[0].ID.String(), asset[0].ID.String(), account[0].ID)
	var buildReq bc.BuildRequest
	err := stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		fmt.Printf("json Unmarshal error.")
	}
	tpl := make([]txbuilder.Template, 1)
	client.Call(context.Background(), "/build-transaction", []*bc.BuildRequest{&buildReq,}, &tpl)
	fmt.Printf("tpl:%v\n", tpl)
}
