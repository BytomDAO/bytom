package wallet

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/tendermint/go-wire/data/base58"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

func TestWalletUpdate(t *testing.T) {
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

	accountManager := account.NewManager(testDB, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, err := hsm.XCreate("test_pub1", "password")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create(nil, []chainkd.XPub{xpub1.XPub}, 1, "testAccount", nil)
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateAddress(nil, testAccount.ID)
	if err != nil {
		t.Fatal(err)
	}

	controlProg.KeyIndex = 1

	utxo := mockUTXO(controlProg)
	_, txData, err := mockTxData(utxo, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)

	reg := asset.NewRegistry(testDB, chain)

	w := mockWallet(testDB, accountManager, reg, chain)

	block := mockSingleBlock(tx)

	txStatus := bc.NewTransactionStatus()
	store.SaveBlock(block, txStatus)

	err = w.AttachBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	want, err := w.GetTransactionsByTxID(tx.ID.String())
	if len(want) != 1 {
		t.Fatal(err)
	}

	wants, err := w.GetTransactionsByTxID("")
	if len(wants) != 1 {
		t.Fatal(err)
	}
}

func TestExportAndImportPrivKey(t *testing.T) {
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

	acntManager := account.NewManager(testDB, chain)
	reg := asset.NewRegistry(testDB, chain)

	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	pwd := "password"
	xpub, err := hsm.XCreate("alias", pwd)
	if err != nil {
		t.Fatal(err)
	}

	w, err := NewWallet(testDB, acntManager, reg, hsm, chain)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	acnt1, err := w.AccountMgr.Create(ctx, []chainkd.XPub{xpub.XPub}, 1, "account-alias", nil)
	if err != nil {
		t.Fatal(err)
	}

	priv, err := w.ExportAccountPrivKey(xpub.XPub, pwd)

	wantPriv, err := hsm.LoadChainKDKey(xpub.XPub, pwd)
	if err != nil {
		t.Fatal(err)
	}
	var hashed [32]byte
	sha3pool.Sum256(hashed[:], wantPriv[:])

	tmp := append(wantPriv[:], hashed[:4]...)
	res := base58.Encode(tmp)

	if res != *priv {
		t.Fatalf("XPrivs should be identical.\nBefore: %v\n After: %v\n", *priv, res)
	}

	rawPriv, err := base58.Decode(*priv)
	if err != nil {
		t.Fatal(err)
	}

	if len(rawPriv) != 68 {
		t.Fatal("invalid private key hash length")
	}

	var xprv [64]byte
	copy(xprv[:], rawPriv[:64])

	_, err = w.ImportAccountPrivKey(xprv, xpub.Alias, pwd, 0, acnt1.Alias)
	if err != pseudohsm.ErrDuplicateKeyAlias {
		t.Fatal(err)
	}

	hsm.XDelete(xpub.XPub, pwd)

	_, err = w.ImportAccountPrivKey(xprv, xpub.Alias, pwd, 0, acnt1.Alias)
	if err != account.ErrDuplicateAlias {
		t.Fatal(err)
	}

	accountInfo := struct {
		AccountInfo string `json:"account_info"`
	}{AccountInfo: acnt1.Alias}

	w.AccountMgr.DeleteAccount(accountInfo)

	acnt2, err := w.ImportAccountPrivKey(xprv, xpub.Alias, pwd, 0, acnt1.Alias)
	if err != nil {
		t.Fatal(err)
	}

	if acnt2.XPub != acnt1.XPubs[0] {
		t.Fatalf("XPubs should be identical.\nBefore: %v\n After: %v\n", acnt1.XPubs[0], acnt2.XPub)
	}
}

func mockUTXO(controlProg *account.CtrlProgram) *account.UTXO {
	utxo := &account.UTXO{}
	utxo.OutputID = bc.Hash{V0: 1}
	utxo.SourceID = bc.Hash{V0: 2}
	utxo.AssetID = *consensus.BTMAssetID
	utxo.Amount = 1000000000
	utxo.SourcePos = 0
	utxo.ControlProgram = controlProg.ControlProgram
	utxo.AccountID = controlProg.AccountID
	utxo.Address = controlProg.Address
	utxo.ControlProgramIndex = controlProg.KeyIndex
	return utxo
}

func mockTxData(utxo *account.UTXO, testAccount *account.Account) (*txbuilder.Template, *types.TxData, error) {
	txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo)
	if err != nil {
		return nil, nil, err
	}

	b := txbuilder.NewBuilder(time.Now())
	b.AddInput(txInput, sigInst)
	out := types.NewTxOutput(*consensus.BTMAssetID, 100, utxo.ControlProgram)
	b.AddOutput(out)
	return b.Build()
}

func mockWallet(walletDB dbm.DB, account *account.Manager, asset *asset.Registry, chain *protocol.Chain) *Wallet {
	return &Wallet{
		DB:             walletDB,
		AccountMgr:     account,
		AssetReg:       asset,
		chain:          chain,
		rescanProgress: make(chan struct{}, 1),
	}
}

func mockSingleBlock(tx *types.Tx) *types.Block {
	return &types.Block{
		BlockHeader: types.BlockHeader{
			Version: 1,
			Height:  1,
			Bits:    2305843009230471167,
		},
		Transactions: []*types.Tx{tx},
	}
}
