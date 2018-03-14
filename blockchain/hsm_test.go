package blockchain

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/blockchain/txdb"
	cfg "github.com/bytom/config"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
)

const dirPath = "pseudohsm/testdata/pseudo"

func TestHSM(t *testing.T) {
	ctx := context.Background()

	dir := tmpManager(t)
	defer os.RemoveAll(dir)

	config := cfg.DefaultConfig()
	tc := dbm.NewDB("txdb", config.DBBackend, dir)
	store := txdb.NewStore(tc)

	var accounts *account.Manager
	var assets *asset.Registry
	// tx pool init
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(bc.Hash{}, store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	walletDB := dbm.NewDB("wallet", config.DBBackend, dir)
	accounts = account.NewManager(walletDB, chain)
	assets = asset.NewRegistry(walletDB, chain)

	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}
	xpub1, err := hsm.XCreate("xpub1", "password")
	if err != nil {
		t.Fatal(err)
	}
	xpub2, err := hsm.XCreate("xpub2", "password")
	if err != nil {
		t.Fatal(err)
	}

	acct1, err := accounts.Create(ctx, []chainkd.XPub{xpub1.XPub}, 1, "acc1", nil)
	if err != nil {
		t.Fatal(err)
	}
	acct2, err := accounts.Create(ctx, []chainkd.XPub{xpub2.XPub}, 1, "acc2", nil)
	if err != nil {
		t.Fatal(err)
	}

	assetDef1 := map[string]interface{}{"foo": 1}
	assetDef2 := map[string]interface{}{"foo": 2}

	asset1, err := assets.Define([]chainkd.XPub{xpub1.XPub}, 1, assetDef1, "foo1", nil)
	if err != nil {
		t.Fatal(err)
	}
	asset2, err := assets.Define([]chainkd.XPub{xpub2.XPub}, 1, assetDef2, "foo2", nil)
	if err != nil {
		t.Fatal(err)
	}

	issue1 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetId: &asset1.AssetID, Amount: 100}))
	issue2 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetId: &asset2.AssetID, Amount: 200}))
	spend1 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset1.AssetID, Amount: 100}, acct1.ID, nil)
	spend2 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset2.AssetID, Amount: 200}, acct2.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{issue1, issue2, spend1, spend2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	//go accounts.ProcessBlocks(ctx)

	err = txbuilder.Sign(ctx, tmpl, nil, "password", func(_ context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
		sigBytes, err := hsm.XSign(xpub, path, data[:], password)
		if err != nil {
			return nil, nil
		}
		return sigBytes, err
	})

	fmt.Printf("###data: %v#####", *tmpl)
	err = hsm.XDelete(xpub1.XPub, "password")
	if err != nil {
		t.Fatal(err)
	}
	err = hsm.XDelete(xpub2.XPub, "password")
	if err != nil {
		t.Fatal(err)
	}
}

func tmpManager(t *testing.T) string {
	d, err := ioutil.TempDir("", "bytom-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	return d
}
