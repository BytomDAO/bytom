package wallet

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
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
	if err := recoveryMgr.AcctResurrect([]chainkd.XPub{xpub.XPub}); err != nil {
		t.Fatal("recovery from XPubs err:", err)
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
	if err := recoveryMgr.AddrResurrect(allAccounts); err != nil {
		t.Fatal("recovery by rescan account err:", err)
	}

	recoveryMgr.FilterRecoveryTxs(MockBlock(txs))
	Accounts, err := acctMgr.ListAccounts("")
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

		if acctMgr.GetContractIndex(acct.ID) != recAcctMgr.GetContractIndex(tmp.ID) {
			t.Fatal("bip32 address index recovery from xpubs err")
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
	recoveryMgr.commitStatusInfo()

	recoveryMgrRestore := newRecoveryManager(testDB, acctMgr)
	recoveryMgrRestore.LoadStatusInfo()

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
