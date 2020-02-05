package wallet

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/blockchain/pseudohsm"
	"github.com/bytom/bytom/blockchain/signers"
	"github.com/bytom/bytom/blockchain/txbuilder"
	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	dbm "github.com/bytom/bytom/database/leveldb"
)

// MockBlock mock a block
func MockBlock(txs []*types.Tx) *types.Block {
	return &types.Block{
		BlockHeader:  types.BlockHeader{Timestamp: uint64(time.Now().Nanosecond())},
		Transactions: txs,
	}
}

func MockSimpleUtxo(index uint64, assetID *bc.AssetID, amount uint64, ctrlProg *account.CtrlProgram) *account.UTXO {
	if ctrlProg == nil {
		ctrlProg = &account.CtrlProgram{
			AccountID:      "",
			Address:        "",
			KeyIndex:       uint64(0),
			ControlProgram: []byte{81},
			Change:         false,
		}
	}

	utxo := &account.UTXO{
		OutputID:            bc.Hash{V0: 1},
		SourceID:            bc.Hash{V0: 1},
		AssetID:             *assetID,
		Amount:              amount,
		SourcePos:           index,
		ControlProgram:      ctrlProg.ControlProgram,
		ControlProgramIndex: ctrlProg.KeyIndex,
		AccountID:           ctrlProg.AccountID,
		Address:             ctrlProg.Address,
		ValidHeight:         0,
	}

	return utxo
}

func AddTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *types.TxOutput {
	out := types.NewTxOutput(assetID, amount, controlProgram)
	return out
}

func BuildTx(baseUtxo *account.UTXO, signer *signers.Signer) (*txbuilder.Template, error) {
	tplBuilder, err := CreateTxBuilder(baseUtxo, signer)
	if err != nil {
		return nil, err
	}

	tpl, _, err := tplBuilder.Build()
	if err != nil {
		return nil, err
	}

	return tpl, nil
}

func CreateTxBuilder(baseUtxo *account.UTXO, signer *signers.Signer) (*txbuilder.TemplateBuilder, error) {
	tplBuilder := txbuilder.NewBuilder(time.Now())
	txOutput := AddTxOutput(baseUtxo.AssetID, 100, baseUtxo.ControlProgram)
	tplBuilder.AddOutput(txOutput)
	return tplBuilder, nil
}

func MockTxsP2PKH(acctMgr *account.Manager, xPub chainkd.XPub, multiTypeAccount bool) ([]*types.Tx, error) {
	txs := []*types.Tx{}
	accts := []*account.Account{}
	for i := uint32(1); i < 32; i = i + 1 + rand.Uint32()%5 {
		alias := fmt.Sprintf("testAccount%d", i)
		deriveRule := signers.BIP0044
		if multiTypeAccount {
			deriveRule = uint8(rand.Uint32() % 2)
		}
		acct, err := account.CreateAccount([]chainkd.XPub{xPub}, 1, alias, uint64(i), deriveRule)
		if err != nil {
			return nil, err
		}

		if err := acctMgr.SaveAccount(acct); err != nil {
			return nil, err
		}

		accts = append(accts, acct)
	}

	for _, acct := range accts {
		for i := uint32(1); i < 256; i = i + 1 + rand.Uint32()%16 {
			controlProg, err := account.CreateCtrlProgram(acct, uint64(i), false)
			if err != nil {
				return nil, err
			}

			if err := acctMgr.SaveControlPrograms(controlProg); err != nil {
				return nil, err
			}

			utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
			tpl, err := BuildTx(utxo, acct.Signer)
			if err != nil {
				return nil, err
			}

			txs = append(txs, tpl.Transaction)
		}
	}

	return txs, nil
}

func TestXPubsRecoveryLock(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub, _, err := hsm.XCreate("test_pub", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr := newRecoveryManager(testDB, acctMgr)
	recoveryMgr.state = newRecoveryState()
	recoveryMgr.state.XPubs = []chainkd.XPub{xpub.XPub}
	recoveryMgr.state.XPubsStatus = newBranchRecoveryState(acctRecoveryWindow)

	recoveryMgr.state.StartTime = time.Now()
	recoveryMgr.commitStatusInfo()

	if err := recoveryMgr.AcctResurrect([]chainkd.XPub{xpub.XPub}); err != nil {
		t.Fatal("TestXPubsRecoveryLock err:", err)
	}

	if err := recoveryMgr.AcctResurrect([]chainkd.XPub{xpub.XPub}); err != errors.Root(ErrRecoveryBusy) {
		t.Fatal("TestXPubsRecoveryLock err:", err)
	}

	if err := recoveryMgr.LoadStatusInfo(); err != errors.Root(ErrRecoveryBusy) {
		t.Fatal("TestXPubsRecoveryLock err:", err)
	}

	recoveryMgr.stopXPubsRec()
	if err := recoveryMgr.LoadStatusInfo(); err != nil {
		t.Fatal("TestXPubsRecoveryLock err:", err)
	}
	recoveryMgr.finished()
	if err := recoveryMgr.AcctResurrect([]chainkd.XPub{xpub.XPub}); err != nil {
		t.Fatal("TestXPubsRecoveryLock err:", err)
	}
}

func TestExtendScanAddresses(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub, _, err := hsm.XCreate("test_pub", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr := newRecoveryManager(testDB, acctMgr)
	acc1 := &account.Account{ID: "testA", Alias: "test1", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub.XPub}, KeyIndex: 1, DeriveRule: signers.BIP0044}}
	acc2 := &account.Account{ID: "testB", Alias: "test2"}
	acc3 := &account.Account{ID: "testC", Alias: "test3", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub.XPub}, KeyIndex: 2, DeriveRule: 3}}
	acc4 := &account.Account{ID: "testD", Alias: "test4", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub.XPub}, KeyIndex: 3, DeriveRule: signers.BIP0032}}

	recoveryMgr.state.stateForScope(acc1)
	recoveryMgr.state.stateForScope(acc3)
	recoveryMgr.state.stateForScope(acc4)

	cases := []struct {
		acct       *account.Account
		err        error
		addressLen uint64
	}{
		{acc1, nil, addrRecoveryWindow * 2},
		{acc2, ErrInvalidAcctID, addrRecoveryWindow * 2},
		{acc3, signers.ErrDeriveRule, addrRecoveryWindow * 2},
		{acc4, nil, addrRecoveryWindow * 3},
	}

	for _, c := range cases {
		if err := recoveryMgr.extendScanAddresses(c.acct.ID, true); err != c.err {
			t.Fatal("extend scan addresses err:", err)
		}

		if err := recoveryMgr.extendScanAddresses(c.acct.ID, false); err != c.err {
			t.Fatal("extend scan addresses err:", err)
		}

		if uint64(len(recoveryMgr.addresses)) != c.addressLen {
			t.Fatalf("extend scan addresses err: len:%d,want:%d", len(recoveryMgr.addresses), c.addressLen)
		}
	}
}

func TestRecoveryFromXPubs(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	recoveryDB := dbm.NewDB("recdb", "leveldb", dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub, _, err := hsm.XCreate("test_pub", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	acctMgr := account.NewManager(testDB, nil)
	txs, err := MockTxsP2PKH(acctMgr, xpub.XPub, false)
	recAcctMgr := account.NewManager(recoveryDB, nil)
	recoveryMgr := newRecoveryManager(recoveryDB, recAcctMgr)

	cases := []struct {
		xPubs []chainkd.XPub
		err   error
	}{
		{[]chainkd.XPub{xpub.XPub}, nil},
		{[]chainkd.XPub{xpub.XPub, xpub.XPub}, signers.ErrDupeXPub},
		{[]chainkd.XPub{}, signers.ErrNoXPubs},
	}

	for _, c := range cases {
		if err := recoveryMgr.AcctResurrect(c.xPubs); errors.Root(err) != c.err {
			t.Fatal("recovery from XPubs err:", err)
		}

		if err != nil {
			recoveryMgr.finished()
			continue
		}
		if err := recoveryMgr.FilterRecoveryTxs(MockBlock(txs)); err != nil {
			t.Fatal("recovery from XPubs err:", err)
		}

		Accounts, err := acctMgr.ListAccounts("")
		if err != nil {
			t.Fatal("recovery from XPubs err:", err)
		}

		for _, acct := range Accounts {
			tmp, err := recAcctMgr.GetAccountByXPubsIndex(acct.XPubs, acct.KeyIndex)
			if err != nil {
				t.Fatal("recovery from XPubs err:", err)
			}

			if tmp == nil {
				t.Fatal("accout recovery from xpubs err:", acct.KeyIndex)
			}

			if acctMgr.GetBip44ContractIndex(acct.ID, true) != recAcctMgr.GetBip44ContractIndex(tmp.ID, true) {
				t.Fatal("bip44 internal address index recovery from xpubs err")
			}

			if acctMgr.GetBip44ContractIndex(acct.ID, false) != recAcctMgr.GetBip44ContractIndex(tmp.ID, false) {
				t.Fatal("bip44 external address index recovery from xpubs err")
			}
		}

		recoveryMgr.finished()
	}
}

func TestRecoveryByRescanAccount(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	recoveryDB := dbm.NewDB("recdb", "leveldb", dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub, _, err := hsm.XCreate("test_pub", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	acctMgr := account.NewManager(testDB, nil)
	txs, err := MockTxsP2PKH(acctMgr, xpub.XPub, true)
	if err != nil {
		t.Fatal("recovery by rescan account err:", err)
	}

	allAccounts, err := acctMgr.ListAccounts("")
	if err != nil {
		t.Fatal("recovery by rescan account err:", err)
	}

	recAcctMgr := account.NewManager(recoveryDB, nil)
	for _, acct := range allAccounts {
		if err := recAcctMgr.SaveAccount(acct); err != nil {
			t.Fatal("recovery by rescan account err:", err)
		}
	}

	recoveryMgr := newRecoveryManager(recoveryDB, recAcctMgr)

	acct := &account.Account{ID: "testA", Alias: "test1", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub.XPub}, KeyIndex: 1, DeriveRule: 3}}

	cases := []struct {
		accounts []*account.Account
		err      error
	}{
		{allAccounts, nil},
		{[]*account.Account{acct}, signers.ErrDeriveRule},
	}

	for _, c := range cases {
		if err := recoveryMgr.AddrResurrect(c.accounts); errors.Root(err) != c.err {
			t.Fatal("recovery by rescan account err:", err)
		}

		if err != nil {
			continue
		}
		recoveryMgr.FilterRecoveryTxs(MockBlock(txs))
		accounts, err := acctMgr.ListAccounts("")
		if err != nil {
			t.Fatal("recovery from XPubs err:", err)
		}

		for _, acct := range accounts {
			tmp, err := recAcctMgr.GetAccountByXPubsIndex(acct.XPubs, acct.KeyIndex)
			if err != nil {
				t.Fatal("recovery from XPubs err:", err)
			}

			if tmp == nil {
				t.Fatal("accout recovery from xpubs err:", acct.KeyIndex)
			}

			if acctMgr.GetBip44ContractIndex(acct.ID, true) != recAcctMgr.GetBip44ContractIndex(tmp.ID, true) {
				t.Fatal("bip44 internal address index recovery from xpubs err")
			}

			if acctMgr.GetBip44ContractIndex(acct.ID, false) != recAcctMgr.GetBip44ContractIndex(tmp.ID, false) {
				t.Fatal("bip44 external address index recovery from xpubs err")
			}

			if acctMgr.GetContractIndex(acct.ID) != recAcctMgr.GetContractIndex(tmp.ID) {
				t.Fatal("bip32 address index recovery from xpubs err")
			}
		}
	}

}

func TestReportFound(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	xpub2, _, err := hsm.XCreate("test_pub2", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr := newRecoveryManager(testDB, acctMgr)
	acc1 := &account.Account{ID: "testA", Alias: "test1", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub1.XPub}, KeyIndex: 1, DeriveRule: signers.BIP0044}}
	acc2 := &account.Account{ID: "testB", Alias: "test2", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub2.XPub}, KeyIndex: 1, DeriveRule: signers.BIP0032}}
	acc3 := &account.Account{ID: "testC", Alias: "test3", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub2.XPub}, KeyIndex: 2, DeriveRule: signers.BIP0044}}

	cp1 := &account.CtrlProgram{AccountID: acc1.ID, Address: "address1", KeyIndex: 10, Change: false}
	cp2 := &account.CtrlProgram{AccountID: acc1.ID, Address: "address1", KeyIndex: 20, Change: true}
	cp3 := &account.CtrlProgram{AccountID: acc2.ID, Address: "address1", KeyIndex: 30, Change: false}
	cp4 := &account.CtrlProgram{AccountID: acc2.ID, Address: "address1", KeyIndex: 40, Change: true}
	cp5 := &account.CtrlProgram{AccountID: acc3.ID, Address: "address1", KeyIndex: 50, Change: false}
	cp6 := &account.CtrlProgram{AccountID: acc3.ID, Address: "address1", KeyIndex: 60, Change: true}

	if err := acctMgr.SaveAccount(acc2); err != nil {
		t.Fatal("ReportFound test err:", err)
	}

	if err := acctMgr.SaveAccount(acc3); err != nil {
		t.Fatal("ReportFound test err:", err)
	}

	recoveryMgr.state.XPubsStatus = newBranchRecoveryState(acctRecoveryWindow)
	recoveryMgr.state.XPubs = []chainkd.XPub{xpub1.XPub}
	recoveryMgr.state.stateForScope(acc1)
	recoveryMgr.state.stateForScope(acc2)
	recoveryMgr.state.stateForScope(acc3)

	cases := []struct {
		acct   *account.Account
		cp     *account.CtrlProgram
		err    error
		status *addressRecoveryState
	}{
		{acc1, cp1, nil,
			&addressRecoveryState{InternalBranch: &branchRecoveryState{addrRecoveryWindow, 1, 1}, ExternalBranch: &branchRecoveryState{addrRecoveryWindow, 139, 11}, Account: acc1}},
		{acc2, cp3, nil,
			&addressRecoveryState{InternalBranch: &branchRecoveryState{addrRecoveryWindow, 1, 1}, ExternalBranch: &branchRecoveryState{addrRecoveryWindow, 159, 31}, Account: acc2}},
		{acc1, cp2, nil,
			&addressRecoveryState{InternalBranch: &branchRecoveryState{addrRecoveryWindow, 149, 21}, ExternalBranch: &branchRecoveryState{addrRecoveryWindow, 139, 11}, Account: acc1}},
		{acc2, cp4, nil,
			&addressRecoveryState{InternalBranch: &branchRecoveryState{addrRecoveryWindow, 169, 41}, ExternalBranch: &branchRecoveryState{addrRecoveryWindow, 159, 31}, Account: acc2}},
		{acc3, cp5, nil,
			&addressRecoveryState{InternalBranch: &branchRecoveryState{addrRecoveryWindow, 1, 1}, ExternalBranch: &branchRecoveryState{addrRecoveryWindow, 179, 51}, Account: acc3}},
		{acc3, cp6, nil,
			&addressRecoveryState{InternalBranch: &branchRecoveryState{addrRecoveryWindow, 189, 61}, ExternalBranch: &branchRecoveryState{addrRecoveryWindow, 179, 51}, Account: acc3}},
	}

	for _, c := range cases {
		if err := recoveryMgr.reportFound(c.acct, c.cp); err != c.err {
			t.Fatal("ReportFound test err:", err, c.acct.ID)
		}

		status, ok := recoveryMgr.state.AccountsStatus[c.acct.ID]
		if !ok {
			t.Fatal("ReportFound test err: can not find status")
		}
		if !reflect.DeepEqual(status, c.status) {
			t.Log(c.status.Account, c.status.InternalBranch, c.status.ExternalBranch)
			t.Log(status.Account, status.InternalBranch, status.ExternalBranch)
			t.Fatal("ReportFound test err: recovery status error")
		}
	}
}

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

	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr := newRecoveryManager(testDB, acctMgr)
	// StatusInit init recovery status manager.
	recoveryMgr.state = newRecoveryState()
	recoveryMgr.state.XPubs = []chainkd.XPub{xpub.XPub}
	recoveryMgr.state.XPubsStatus = newBranchRecoveryState(acctRecoveryWindow)

	recoveryMgr.state.StartTime = time.Now()
	if err := recoveryMgr.LoadStatusInfo(); err != nil {
		t.Fatal("TestLoadStatusInfo err:", err)
	}

	recoveryMgr.commitStatusInfo()

	recoveryMgrRestore := newRecoveryManager(testDB, acctMgr)
	if err := recoveryMgrRestore.LoadStatusInfo(); err != nil {
		t.Fatal("TestLoadStatusInfo err:", err)
	}

	if !reflect.DeepEqual(recoveryMgrRestore.state.XPubsStatus, recoveryMgr.state.XPubsStatus) {
		t.Fatalf("TestLoadStatusInfo XPubsStatus reload err")
	}

	if !reflect.DeepEqual(recoveryMgrRestore.state.XPubs, recoveryMgr.state.XPubs) {
		t.Fatalf("TestLoadStatusInfo XPubs reload err")
	}

	if !reflect.DeepEqual(recoveryMgrRestore.state.AccountsStatus, recoveryMgr.state.AccountsStatus) {
		t.Fatalf("TestLoadStatusInfo AccountsStatus reload err")
	}

	if !recoveryMgrRestore.state.StartTime.Equal(recoveryMgr.state.StartTime) {
		t.Fatalf("TestLoadStatusInfo StartTime reload err")
	}

	acct := &account.Account{ID: "testA", Alias: "test1", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub.XPub}, KeyIndex: 1, DeriveRule: 3}}
	recoveryMgr.state.AccountsStatus[acct.ID] = newAddressRecoveryState(addrRecoveryWindow, acct)
	if err := recoveryMgr.commitStatusInfo(); err != nil {
		t.Fatal("TestLoadStatusInfo err:", err)
	}
	if err := recoveryMgr.LoadStatusInfo(); err == nil {
		t.Fatal("TestLoadStatusInfo err")
	}

	recoveryMgr.state = nil
	if err := recoveryMgr.commitStatusInfo(); err != nil {
		t.Fatal("TestLoadStatusInfo err:", err)
	}

	if err := recoveryMgr.LoadStatusInfo(); err == nil {
		t.Fatal("TestLoadStatusInfo err")
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

	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr := newRecoveryManager(testDB, acctMgr)
	if !recoveryMgr.tryStartXPubsRec() {
		t.Fatal("recovery manager try lock test err")
	}

	if recoveryMgr.tryStartXPubsRec() {
		t.Fatal("recovery manager relock test err")
	}

	recoveryMgr.stopXPubsRec()

	if !recoveryMgr.tryStartXPubsRec() {
		t.Fatal("recovery manager try lock test err")
	}
}

func TestStateForScope(t *testing.T) {
	state := newRecoveryState()
	acc1 := &account.Account{ID: "test1", Alias: "testA"}
	state.stateForScope(acc1)
	if !reflect.DeepEqual(state.AccountsStatus[acc1.ID].Account, acc1) {
		t.Fatal("state for scope test err")
	}

	acc2 := &account.Account{ID: "test1", Alias: "testB"}
	state.stateForScope(acc2)

	if reflect.DeepEqual(state.AccountsStatus[acc2.ID].Account, acc2) {
		t.Fatal("state for scope test err")
	}

	acc3 := &account.Account{ID: "test2", Alias: "testC"}
	state.stateForScope(acc3)
	if !reflect.DeepEqual(state.AccountsStatus[acc3.ID].Account, acc3) {
		t.Fatal("state for scope test err")
	}
}

func bip44ContractIndexKey(accountID string, change bool) []byte {
	contractIndexPrefix := []byte("ContractIndex")
	key := append(contractIndexPrefix, accountID...)
	if change {
		return append(key, []byte{1}...)
	}
	return append(key, []byte{0}...)
}

func TestContractIndexResidue(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	contractIndexResidue := uint64(5)
	acctMgr := account.NewManager(testDB, nil)
	recoveryMgr := newRecoveryManager(testDB, acctMgr)
	acct := &account.Account{ID: "testA", Alias: "test1", Signer: &signers.Signer{XPubs: []chainkd.XPub{xpub1.XPub}, KeyIndex: 1, DeriveRule: signers.BIP0044}}

	cp1 := &account.CtrlProgram{AccountID: acct.ID, Address: "address1", KeyIndex: 10, Change: false}

	setContractIndexKey := func(acctMgr *account.Manager, accountID string, change bool) {
		testDB.Set(bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(contractIndexResidue))
	}

	delAccount := func(acctMgr *account.Manager, accountID string, change bool) {
		acctMgr.DeleteAccount(accountID)
	}

	recoveryMgr.state.XPubsStatus = newBranchRecoveryState(acctRecoveryWindow)
	recoveryMgr.state.XPubs = []chainkd.XPub{xpub1.XPub}
	recoveryMgr.state.stateForScope(acct)

	cases := []struct {
		acct       *account.Account
		cp         *account.CtrlProgram
		preProcess func(acctMgr *account.Manager, accountID string, change bool)
		err        error
		wantCPNum  uint64
	}{
		{acct, cp1, setContractIndexKey, nil, 5},
		{acct, cp1, delAccount, nil, 10},
	}

	for _, c := range cases {
		if c.preProcess != nil {
			c.preProcess(acctMgr, c.acct.ID, c.cp.Change)
		}

		if err := acctMgr.SaveAccount(acct); err != nil {
			t.Fatal("ReportFound test err:", err)
		}

		if err := recoveryMgr.reportFound(c.acct, c.cp); err != c.err {
			t.Fatal("ContractIndexResidue test err:", err, c.acct.ID)
		}
		cps, err := acctMgr.ListControlProgram()
		if err != nil {
			t.Fatal("list control program err:", err)
		}

		cpNum := uint64(0)
		for _, cp := range cps {
			if cp.Address == "" || cp.AccountID != c.acct.ID {
				continue
			}
			cpNum++
		}

		if cpNum != c.wantCPNum {
			t.Fatal("Test contract index residue cp num err want:", c.wantCPNum, " got:", cpNum)
		}
	}
}
