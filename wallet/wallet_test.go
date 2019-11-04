package wallet

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/asset"
	"github.com/bytom/bytom/blockchain/pseudohsm"
	"github.com/bytom/bytom/blockchain/signers"
	"github.com/bytom/bytom/blockchain/txbuilder"
	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func TestEncodeDecodeGlobalTxIndex(t *testing.T) {
	want := &struct {
		BlockHash bc.Hash
		Position  uint64
	}{
		BlockHash: bc.NewHash([32]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}),
		Position:  1,
	}

	globalTxIdx := calcGlobalTxIndex(&want.BlockHash, want.Position)
	blockHashGot, positionGot := parseGlobalTxIdx(globalTxIdx)
	if *blockHashGot != want.BlockHash {
		t.Errorf("blockHash mismatch. Get: %v. Expect: %v", *blockHashGot, want.BlockHash)
	}

	if positionGot != want.Position {
		t.Errorf("position mismatch. Get: %v. Expect: %v", positionGot, want.Position)
	}
}

func TestWalletVersion(t *testing.T) {
	// prepare wallet
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	dispatcher := event.NewDispatcher()
	w := mockWallet(testDB, nil, nil, nil, dispatcher, false)

	// legacy status test case
	type legacyStatusInfo struct {
		WorkHeight uint64
		WorkHash   bc.Hash
		BestHeight uint64
		BestHash   bc.Hash
	}
	rawWallet, err := json.Marshal(legacyStatusInfo{})
	if err != nil {
		t.Fatal("Marshal legacyStatusInfo")
	}

	w.DB.Set(walletKey, rawWallet)
	rawWallet = w.DB.Get(walletKey)
	if rawWallet == nil {
		t.Fatal("fail to load wallet StatusInfo")
	}

	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
		t.Fatal(err)
	}

	if err := w.checkWalletInfo(); err != errWalletVersionMismatch {
		t.Fatal("fail to detect legacy wallet version")
	}

	// lower wallet version test case
	lowerVersion := StatusInfo{Version: currentVersion - 1}
	rawWallet, err = json.Marshal(lowerVersion)
	if err != nil {
		t.Fatal("save wallet info")
	}

	w.DB.Set(walletKey, rawWallet)
	rawWallet = w.DB.Get(walletKey)
	if rawWallet == nil {
		t.Fatal("fail to load wallet StatusInfo")
	}

	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
		t.Fatal(err)
	}

	if err := w.checkWalletInfo(); err != errWalletVersionMismatch {
		t.Fatal("fail to detect expired wallet version")
	}
}

func TestWalletUpdate(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	store := database.NewStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)

	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	accountManager := account.NewManager(testDB, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	controlProg.KeyIndex = 1

	reg := asset.NewRegistry(testDB, chain)
	asset, err := reg.Define([]chainkd.XPub{xpub1.XPub}, 1, nil, 0, "TESTASSET", nil)
	if err != nil {
		t.Fatal(err)
	}

	utxos := []*account.UTXO{}
	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)
	utxos = append(utxos, btmUtxo)
	OtherUtxo := mockUTXO(controlProg, &asset.AssetID)
	utxos = append(utxos, OtherUtxo)

	_, txData, err := mockTxData(utxos, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)
	block := mockSingleBlock(tx)
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatus.SetStatus(1, false)
	store.SaveBlock(block, txStatus)

	w := mockWallet(testDB, accountManager, reg, chain, dispatcher, true)
	err = w.AttachBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.GetTransactionByTxID(tx.ID.String()); err != nil {
		t.Fatal(err)
	}

	wants, err := w.GetTransactions("")
	if len(wants) != 1 {
		t.Fatal(err)
	}

	if wants[0].ID != tx.ID {
		t.Fatal("account txID mismatch")
	}

	for position, tx := range block.Transactions {
		get := w.DB.Get(calcGlobalTxIndexKey(tx.ID.String()))
		bh := block.BlockHeader.Hash()
		expect := calcGlobalTxIndex(&bh, uint64(position))
		if !reflect.DeepEqual(get, expect) {
			t.Fatalf("position#%d: compare retrieved globalTxIdx err", position)
		}
	}
}

func TestRescanWallet(t *testing.T) {
	// prepare wallet & db
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	store := database.NewStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	statusInfo := StatusInfo{
		Version:  currentVersion,
		WorkHash: bc.Hash{V0: 0xff},
	}
	rawWallet, err := json.Marshal(statusInfo)
	if err != nil {
		t.Fatal("save wallet info")
	}

	w := mockWallet(testDB, nil, nil, chain, dispatcher, false)
	w.DB.Set(walletKey, rawWallet)
	rawWallet = w.DB.Get(walletKey)
	if rawWallet == nil {
		t.Fatal("fail to load wallet StatusInfo")
	}

	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
		t.Fatal(err)
	}

	// rescan wallet
	if err := w.loadWalletInfo(); err != nil {
		t.Fatal(err)
	}

	block := config.GenesisBlock()
	if w.status.WorkHash != block.Hash() {
		t.Fatal("reattach from genesis block")
	}
}

func TestMemPoolTxQueryLoop(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)

	store := database.NewStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)

	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	accountManager := account.NewManager(testDB, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	controlProg.KeyIndex = 1

	reg := asset.NewRegistry(testDB, chain)
	asset, err := reg.Define([]chainkd.XPub{xpub1.XPub}, 1, nil, 0, "TESTASSET", nil)
	if err != nil {
		t.Fatal(err)
	}

	utxos := []*account.UTXO{}
	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)
	utxos = append(utxos, btmUtxo)
	OtherUtxo := mockUTXO(controlProg, &asset.AssetID)
	utxos = append(utxos, OtherUtxo)

	_, txData, err := mockTxData(utxos, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)
	//block := mockSingleBlock(tx)
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	w, err := NewWallet(testDB, accountManager, reg, hsm, chain, dispatcher, false)
	go w.memPoolTxQueryLoop()
	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: protocol.MsgNewTx}})
	time.Sleep(time.Millisecond * 10)
	if _, err = w.GetUnconfirmedTxByTxID(tx.ID.String()); err != nil {
		t.Fatal("disaptch new tx msg error:", err)
	}
	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: protocol.MsgRemoveTx}})
	time.Sleep(time.Millisecond * 10)
	txs, err := w.GetUnconfirmedTxs(testAccount.ID)
	if err != nil {
		t.Fatal("get unconfirmed tx error:", err)
	}

	if len(txs) != 0 {
		t.Fatal("disaptch remove tx msg error")
	}

	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: 2}})
}

func mockUTXO(controlProg *account.CtrlProgram, assetID *bc.AssetID) *account.UTXO {
	utxo := &account.UTXO{}
	utxo.OutputID = bc.Hash{V0: 1}
	utxo.SourceID = bc.Hash{V0: 2}
	utxo.AssetID = *assetID
	utxo.Amount = 1000000000
	utxo.SourcePos = 0
	utxo.ControlProgram = controlProg.ControlProgram
	utxo.AccountID = controlProg.AccountID
	utxo.Address = controlProg.Address
	utxo.ControlProgramIndex = controlProg.KeyIndex
	return utxo
}

func mockTxData(utxos []*account.UTXO, testAccount *account.Account) (*txbuilder.Template, *types.TxData, error) {
	tplBuilder := txbuilder.NewBuilder(time.Now())

	for _, utxo := range utxos {
		txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo)
		if err != nil {
			return nil, nil, err
		}
		tplBuilder.AddInput(txInput, sigInst)

		out := &types.TxOutput{}
		if utxo.AssetID == *consensus.BTMAssetID {
			out = types.NewTxOutput(utxo.AssetID, 100, utxo.ControlProgram)
		} else {
			out = types.NewTxOutput(utxo.AssetID, utxo.Amount, utxo.ControlProgram)
		}
		tplBuilder.AddOutput(out)
	}

	return tplBuilder.Build()
}

func mockWallet(walletDB dbm.DB, account *account.Manager, asset *asset.Registry, chain *protocol.Chain, dispatcher *event.Dispatcher, txIndexFlag bool) *Wallet {
	wallet := &Wallet{
		DB:              walletDB,
		AccountMgr:      account,
		AssetReg:        asset,
		chain:           chain,
		RecoveryMgr:     newRecoveryManager(walletDB, account),
		eventDispatcher: dispatcher,
		TxIndexFlag:     txIndexFlag,
	}
	wallet.txMsgSub, _ = wallet.eventDispatcher.Subscribe(protocol.TxMsgEvent{})
	return wallet
}

func mockSingleBlock(tx *types.Tx) *types.Block {
	return &types.Block{
		BlockHeader: types.BlockHeader{
			Version: 1,
			Height:  1,
			Bits:    2305843009230471167,
		},
		Transactions: []*types.Tx{config.GenesisTx(), tx},
	}
}
