package account

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/testutil"
)

func TestCreateAccountWithUppercase(t *testing.T) {
	m := mockAccountManager(t)
	alias := "UPPER"
	account, err := m.Create(nil, []chainkd.XPub{testutil.TestXPub}, 1, alias)

	if err != nil {
		t.Fatal(err)
	}

	if account.Alias != strings.ToLower(alias) {
		t.Fatal("created account alias should be lowercase")
	}
}

func TestCreateAccountWithSpaceTrimed(t *testing.T) {
	m := mockAccountManager(t)
	alias := " with space "
	account, err := m.Create(nil, []chainkd.XPub{testutil.TestXPub}, 1, alias)

	if err != nil {
		t.Fatal(err)
	}

	if account.Alias != strings.TrimSpace(alias) {
		t.Fatal("created account alias should be lowercase")
	}
}

func TestCreateAccount(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.FindByID(ctx, account.ID)
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

	_, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias")
	if errors.Root(err) != ErrDuplicateAlias {
		t.Errorf("expected %s when reusing an alias, got %v", ErrDuplicateAlias, err)
	}
}

func TestDeleteAccount(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account1, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias1")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account2, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias2")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if err = m.DeleteAccount(account1.Alias); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.FindByID(ctx, account1.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}

	if err = m.DeleteAccount(account2.ID); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err = m.FindByID(ctx, account2.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}
}

func TestFindByID(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()
	account := m.createTestAccount(ctx, t, "", nil)

	found, err := m.FindByID(ctx, account.ID)
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
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	return NewManager(testDB, chain)
}

func (m *Manager) createTestAccount(ctx context.Context, t testing.TB, alias string, tags map[string]interface{}) *Account {
	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account

}
