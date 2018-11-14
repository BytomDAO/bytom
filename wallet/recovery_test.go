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
	for i := uint32(0); i < 32; i = i + 1 + rand.Uint32()%5 {
		alias := fmt.Sprintf("testAccount%d", i)
		deriveRule := signers.BIP0044
		if multiTypeAccount {
			deriveRule = uint8(rand.Uint32() % 2)
		}
		acct, err := acctMgr.Create([]chainkd.XPub{xPub}, 1, alias, deriveRule)
		if err != nil {
			return nil, err
		}
		accts = append(accts, acct)
	}

	for _, acct := range accts {
		num := rand.Uint32() % 256
		for i := uint32(0); i < num; i = i + 1 + rand.Uint32()%16 {
			controlProg, err := acctMgr.CreateAddress(acct.ID, false)
			if err != nil {
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
