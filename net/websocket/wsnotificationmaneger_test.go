package websocket

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/event"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/test"
	dbm "github.com/tendermint/tmlibs/db"
)

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

func TestMemPoolTxQueryLoop(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)

	store := leveldb.NewStore(testDB)
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

	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)

	_, txData, err := test.MockTx(btmUtxo, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)
	txD := &protocol.TxDesc{
		Tx:         tx,
		StatusFail: false,
		Weight:     tx.SerializedSize,
		Height:     1,
		Fee:        1,
	}
	nm := NewWsNotificationManager(1, 1, chain, dispatcher)
	nm.txMsgSub, _ = nm.eventDispatcher.Subscribe(protocol.TxMsgEvent{})

	go nm.memPoolTxQueryLoop()
	nm.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: txD, MsgType: protocol.MsgNewTx}})
	result := make(chan bool, 1)
	go readFromCh(nm.queueNotification, txD, result)
	ok, _ := <-result
	if !ok {
		t.Fatal("test error")
	}
}

func readFromCh(in <-chan interface{}, wantTxDesc *protocol.TxDesc, result chan bool) chan bool {
	for {
		select {
		case n, _ := <-in:
			obj := n.(*notificationTxDescAcceptedByMempool)
			if !reflect.DeepEqual(obj, (*notificationTxDescAcceptedByMempool)(wantTxDesc)) {
				result <- false
			} else {
				result <- true
			}
		}
	}
}
