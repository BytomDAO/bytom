package wallet

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/crypto/ed25519/chainkd"
)

func TestLoadStatusInfo(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub, _, err := hsm.XCreate("test_pub", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	recoveryMgr := newRecoveryManager(testDB)
	recoveryMgr.statusInit([]chainkd.XPub{xpub.XPub})
	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr.resurrect(acctMgr)
	recoveryMgr.state.StartTime = time.Now()
	recoveryMgr.commitStatusInfo()

	recoveryMgrRestore := newRecoveryManager(testDB)
	recoveryMgrRestore.loadStatusInfo(acctMgr)

	if !reflect.DeepEqual(recoveryMgrRestore.state.XPubsStatus, recoveryMgr.state.XPubsStatus) {
		t.Fatalf("testLoadStatusInfo XPubsStatus reload err")
	}

	if !reflect.DeepEqual(recoveryMgrRestore.state.XPubs, recoveryMgr.state.XPubs) {
		t.Fatalf("testLoadStatusInfo XPubs reload err")
	}

	if !reflect.DeepEqual(recoveryMgrRestore.state.AccountsStatus, recoveryMgr.state.AccountsStatus) {
		t.Fatalf("testLoadStatusInfo AccountsStatus reload err")
	}

	if !recoveryMgrRestore.state.StartTime.Equal(recoveryMgr.state.StartTime) {
		t.Fatalf("testLoadStatusInfo StartTime reload err")
	}
}

func TestLock(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	recoveryMgr := newRecoveryManager(testDB)
	if !recoveryMgr.tryLock() {
		t.Fatal("recovery manager try lock test err")
	}

	if recoveryMgr.tryLock() {
		t.Fatal("recovery manager relock test err")
	}

	recoveryMgr.unLock()

	if !recoveryMgr.tryLock() {
		t.Fatal("recovery manager try lock test err")
	}
}
