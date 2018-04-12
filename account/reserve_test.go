package account

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/state"
)

func TestCancelReservation(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	chain, err := mockChain(testDB)
	if err != nil {
		t.Fatal(err)
	}

	accountManager := NewManager(testDB, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, err := hsm.XCreate("test_pub", "password")
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

	utxo := mockUTXO(controlProg)

	batch := testDB.NewBatch()

	utxoE := struct {
		hash      bc.Hash
		utxoEntry *storage.UtxoEntry
		exist     bool
	}{
		hash:      utxo.OutputID,
		utxoEntry: storage.NewUtxoEntry(true, 0, false),
		exist:     true,
	}

	view := state.NewUtxoViewpoint()
	view.Entries[utxoE.hash] = utxoE.utxoEntry

	leveldb.SaveUtxoView(batch, view)
	batch.Write()

	utxoDB := newReserver(chain, testDB)

	batch = utxoDB.db.NewBatch()

	data, err := json.Marshal(utxo)
	if err != nil {
		t.Fatal(err)
	}

	batch.Set(StandardUTXOKey(utxo.OutputID), data)
	batch.Write()

	outid := utxo.OutputID

	ctx := context.Background()
	res, err := utxoDB.ReserveUTXO(ctx, outid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the UTXO is reserved.
	_, err = utxoDB.ReserveUTXO(ctx, outid, nil, time.Now())
	if err != ErrReserved {
		t.Fatalf("got=%s want=%s", err, ErrReserved)
	}

	// Cancel the reservation.
	err = utxoDB.Cancel(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Reserving again should succeed.
	_, err = utxoDB.ReserveUTXO(ctx, outid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func mockChain(testDB dbm.DB) (*protocol.Chain, error) {
	store := leveldb.NewStore(testDB)
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func mockUTXO(controlProg *CtrlProgram) *UTXO {
	utxo := &UTXO{}
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
