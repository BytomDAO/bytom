package asset

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestDefineAsset(t *testing.T) {
	ctx := context.Background()
	reg := mockNewRegistry(t)
	asset, err := reg.Define(ctx, []chainkd.XPub{testutil.TestXPub}, 1, nil, "asset-alias", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := reg.findByID(ctx, &asset.AssetID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected asset %s to be recorded as %s", asset, found)
	}
}

func TestFindAssetByID(t *testing.T) {
	ctx := context.Background()
	reg := mockNewRegistry(t)
	keys := []chainkd.XPub{testutil.TestXPub}
	asset, err := reg.Define(ctx, keys, 1, nil, "", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)

	}
	found, err := reg.findByID(ctx, &asset.AssetID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected %v and %v to match", asset, found)
	}

}

func mockChain(testDB dbm.DB) (*protocol.Chain, error) {
	store := txdb.NewStore(testDB)
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
