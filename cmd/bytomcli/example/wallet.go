package example

import (
    "context"
    stdjson "encoding/json"
    "encoding/hex"
    "fmt"
    _"time"

    bc "github.com/bytom/blockchain"
    "github.com/bytom/blockchain/query"
    "github.com/bytom/blockchain/rpc"
    "github.com/bytom/blockchain/txbuilder"

    "github.com/bytom/crypto/ed25519/chainkd"
    _"github.com/bytom/encoding/json"
)

// TO DO: issue a asset to a account.
func WalletTest(client *rpc.Client, args []string) {
    // Create Account.
    /*
    fmt.Printf("To create Account:\n")
    xprv, _ := chainkd.NewXPrv(nil)
    xpub := xprv.XPub()
    fmt.Printf("xprv_account:%v\n", xprv)
    fmt.Printf("xpub_account:%v\n", xpub)
    */
    xpub_str := "7ed0a605bf967c6329c29dff17b924ddfb05e1c09d8dc993309919e82a817a87f11b4f2b3464eb7303b74ec0a41604f3691bfd39bfd59c8ab6f85333bc7d127f"
    xpub_account := new(chainkd.XPub)
    data, err := hex.DecodeString(xpub_str)
    if err != nil {
        fmt.Printf("error:  %v", err)
    }
    copy(xpub_account[:], data)

    type Ins struct {
        RootXPubs   []chainkd.XPub `json:"root_xpubs"`
        Quorum      int
        Alias       string
        Tags        map[string]interface{}
        ClientToken string `json:"client_token"`
    }
    var ins Ins
    ins.RootXPubs = []chainkd.XPub{*xpub_account}
    ins.Quorum = 1
    ins.Alias = "alice"
    ins.Tags = map[string]interface{}{"test_tag": "v0"}
    ins.ClientToken = "account"
    account := make([]query.AnnotatedAccount, 1)
    client.Call(context.Background(), "/create-account", &[]Ins{ins}, &account)
    fmt.Printf("account:%v\n", account)

    // Create Asset.
    fmt.Printf("To create Asset:\n")
    xprv_asset, _ := chainkd.NewXPrv(nil)
    xpub_asset := xprv_asset.XPub()
    fmt.Printf("xprv_asset:%v\n", xprv_asset)
    fmt.Printf("xpub_asset:%v\n", xpub_asset)
    type Ins_asset struct {
        RootXPubs   []chainkd.XPub `json:"root_xpubs"`
        Quorum      int
        Alias       string
        Tags        map[string]interface{}
        Definition  map[string]interface{}
        ClientToken string `json:"client_token"`
    }
    var ins_asset Ins_asset
    ins_asset.RootXPubs = []chainkd.XPub{xpub_asset}
    ins_asset.Quorum = 1
    ins_asset.Alias = "gold"
    ins_asset.Tags = map[string]interface{}{"test_tag": "v0"}
    ins_asset.Definition = map[string]interface{}{"test_definition": "v0"}
    ins_asset.ClientToken = "asset"
    asset := make([]query.AnnotatedAsset, 1)
    client.Call(context.Background(), "/create-asset", &[]Ins_asset{ins_asset}, &asset)
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
    err = stdjson.Unmarshal([]byte(buildReqStr), &buildReq)
    if err != nil {
        fmt.Printf("json Unmarshal error.")
    }

    tpl := make([]txbuilder.Template, 1)
    client.Call(context.Background(), "/build-transaction", []*bc.BuildRequest{&buildReq}, &tpl)
    fmt.Printf("tpl:%v\n", tpl)

    // sign-transaction
    var response interface{}
    type Ins_tx struct {
        Auth string
        Txs   []*txbuilder.Template `json:"transactions"`
        XPubs []chainkd.XPub        `json:"xpubs"`
    }
    var tx Ins_tx
    tx.Auth = "langyu"
    tx.Txs = []*txbuilder.Template{&tpl[0]}
    tx.XPubs = []chainkd.XPub{*xpub_account}

    client.Call(context.Background(), "/sign-transactions", &tx,  &response)
   /*err = txbuilder.Sign(context.Background(), &tpl[0], []chainkd.XPub{xprv_asset.XPub()}, func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
        derived := xprv_asset.Derive(path)
        return derived.Sign(data[:]), nil
    })
    */
    fmt.Printf("response %v\n", response)
    if err != nil {
        fmt.Printf("sign-transaction error. err:%v\n", err)
    }
    fmt.Printf("sign tpl:%v\n", tpl[0])
    fmt.Printf("sign tpl's SigningInstructions:%v\n", tpl[0].SigningInstructions[0])
    fmt.Printf("SigningInstructions's SignatureWitnesses:%v\n", tpl[0].SigningInstructions[0].SignatureWitnesses[0])

    // submit-transaction
    /*
    var submitResponse interface{}
    submitArg := bc.SubmitArg{tpl, json.Duration{time.Duration(1000000)}, "none"}
    client.Call(context.Background(), "/submit-transaction", submitArg, &submitResponse)
    fmt.Printf("submit transaction:%v\n", submitResponse)
    */
}