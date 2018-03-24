package account

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestCreateAccount(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias", nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.findByID(ctx, account.ID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected account %v to be recorded as %v", account, found)
	}
}

func TestCreateAccountReusedAlias(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()
	m.createTestAccount(ctx, t, "test-alias", nil)

	_, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias", nil)
	if errors.Root(err) != ErrDuplicateAlias {
		t.Errorf("expected %s when reusing an alias, got %v", ErrDuplicateAlias, err)
	}
}

func TestDeleteAccount(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account1, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias1", nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account2, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias2", nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	cases := []struct {
		AccountInfo string `json:"account_info"`
	}{
		{AccountInfo: account1.Alias},
		{AccountInfo: account2.ID},
	}

	if err = m.DeleteAccount(cases[0]); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.findByID(ctx, account1.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}

	if err = m.DeleteAccount(cases[1]); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err = m.findByID(ctx, account2.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}
}

func TestUpdateAccountTags(t *testing.T) {
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

	m := NewManager(testDB, chain)
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

func TestFindByID(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()
	account := m.createTestAccount(ctx, t, "", nil)

	found, err := m.findByID(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindByAlias(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()
	account := m.createTestAccount(ctx, t, "some-alias", nil)

	found, err := m.FindByAlias(ctx, "some-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func mockAccountManager(t *testing.T) *Manager {
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

	return NewManager(testDB, chain)
}

func (m *Manager) createTestAccount(ctx context.Context, t testing.TB, alias string, tags map[string]interface{}) *Account {
	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, alias, tags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account

}
