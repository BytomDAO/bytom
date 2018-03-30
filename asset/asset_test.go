package asset

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"strings"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/consensus"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestDefineAssetWithLowercase(t *testing.T) {
	reg := mockNewRegistry(t)
	alias := "lower"
	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, alias, nil)
	if err != nil {
		t.Fatal(err)
	}
	if *asset.Alias != strings.ToUpper(alias) {
		t.Fatal("created asset alias should be uppercase")
	}
}

func TestDefineAssetWithSpaceTrimed(t *testing.T) {
	reg := mockNewRegistry(t)
	alias := " WITH SPACE "
	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, alias, nil)
	if err != nil {
		t.Fatal(err)
	}
	if *asset.Alias != strings.TrimSpace(alias) {
		t.Fatal("created asset alias should be uppercase")
	}
}

func TestDefineAsset(t *testing.T) {
	ctx := context.Background()
	reg := mockNewRegistry(t)
	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, "asset-alias", nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := reg.FindByID(ctx, &asset.AssetID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected asset %v to be recorded as %v", asset, found)
	}
}

func TestDefineBtmAsset(t *testing.T) {
	reg := mockNewRegistry(t)
	_, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, consensus.BTMAlias, nil)
	if err == nil {
		testutil.FatalErr(t, err)
	}
}

func TestFindAssetByID(t *testing.T) {
	ctx := context.Background()
	reg := mockNewRegistry(t)
	keys := []chainkd.XPub{testutil.TestXPub}
	asset, err := reg.Define(keys, 1, nil, "", nil)
	if err != nil {
		testutil.FatalErr(t, err)

	}
	found, err := reg.FindByID(ctx, &asset.AssetID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected %v and %v to match", asset, found)
	}
}

func TestUpdateAssetTags(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	store := leveldb.NewStore(testDB)
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(bc.Hash{}, store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry(testDB, chain)
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

func mockChain(testDB dbm.DB) (*protocol.Chain, error) {
	store := leveldb.NewStore(testDB)
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(bc.Hash{}, store, txPool)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func mockNewRegistry(t *testing.T) *Registry {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	chain, err := mockChain(testDB)
	if err != nil {
		t.Fatal(err)
	}

	return NewRegistry(testDB, chain)
}
