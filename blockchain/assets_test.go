package blockchain

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestUpdateAssetTags(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	store := txdb.NewStore(testDB)
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(bc.Hash{}, store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	reg := asset.NewRegistry(testDB, chain)
	ctx := context.Background()

	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, "asset-alias",
		map[string]interface{}{
			"test_tag": "v0",
		})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Update by ID
	wantTags := map[string]interface{}{
		"test_tag": "v1",
	}

	if reg.UpdateTags(ctx, asset.AssetID.String(), wantTags) != nil {
		testutil.FatalErr(t, err)
	}

	asset1, err := reg.FindByAlias(ctx, *asset.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotTags := asset1.Tags
	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", gotTags, wantTags)
	}

	// Update by alias
	wantTags = map[string]interface{}{
		"test_tag": "v2",
	}

	if reg.UpdateTags(ctx, *asset.Alias, wantTags) != nil {
		testutil.FatalErr(t, err)
	}

	asset2, err := reg.FindByAlias(ctx, *asset.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotTags = asset2.Tags
	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", gotTags, wantTags)
	}
}
