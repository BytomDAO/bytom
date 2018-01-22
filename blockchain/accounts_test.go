package blockchain

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestUpdateAccountTags(t *testing.T) {
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

	m := account.NewManager(testDB, chain)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "account-alias",
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

	if m.UpdateTags(ctx, account.ID, wantTags) != nil {
		testutil.FatalErr(t, err)
	}

	account1, err := m.FindByAlias(ctx, account.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotTags := account1.Tags
	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", gotTags, wantTags)
	}

	// Update by alias
	wantTags = map[string]interface{}{
		"test_tag": "v2",
	}

	if m.UpdateTags(ctx, account.Alias, wantTags) != nil {
		testutil.FatalErr(t, err)
	}

	account2, err := m.FindByAlias(ctx, account.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotTags = account2.Tags
	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Fatalf("tags:\ngot:  %v\nwant: %v", gotTags, wantTags)
	}
}
