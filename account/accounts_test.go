package account

import (
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
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias)

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
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias)

	if err != nil {
		t.Fatal(err)
	}

	if account.Alias != strings.TrimSpace(alias) {
		t.Fatal("created account alias should be lowercase")
	}

	nilAccount, err := m.FindByAlias(alias)
	if nilAccount != nil {
		t.Fatal("expected nil")
	}

	target, err := m.FindByAlias(strings.ToLower(strings.TrimSpace(alias)))
	if target == nil {
		t.Fatal("expected Account, but got nil")
	}
}

func TestCreateAccount(t *testing.T) {
	m := mockAccountManager(t)
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.FindByID(account.ID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected account %v to be recorded as %v", account, found)
	}
}

func TestCreateAccountReusedAlias(t *testing.T) {
	m := mockAccountManager(t)
	m.createTestAccount(t, "test-alias", nil)

	_, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias")
	if errors.Root(err) != ErrDuplicateAlias {
		t.Errorf("expected %s when reusing an alias, got %v", ErrDuplicateAlias, err)
	}
}

func TestDeleteAccount(t *testing.T) {
	m := mockAccountManager(t)

	account1, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias1")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account2, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias2")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if err = m.DeleteAccount(account1.Alias); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.FindByID(account1.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}

	if err = m.DeleteAccount(account2.ID); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err = m.FindByID(account2.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}
}

func TestFindByID(t *testing.T) {
	m := mockAccountManager(t)
	account := m.createTestAccount(t, "", nil)

	found, err := m.FindByID(account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindByAlias(t *testing.T) {
	m := mockAccountManager(t)
	account := m.createTestAccount(t, "some-alias", nil)

	found, err := m.FindByAlias("some-alias")
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

func (m *Manager) createTestAccount(t testing.TB, alias string, tags map[string]interface{}) *Account {
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account

}
