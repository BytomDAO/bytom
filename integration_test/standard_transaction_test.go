package integrationTest

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/blockchain/txdb"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm"
)

func TestP2PKH(t *testing.T) {
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

	accountManager := account.NewManager(testDB, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub, err := hsm.XCreate("test_pub", "password")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create(nil, []chainkd.XPub{xpub.XPub}, 1, "testAccount", nil, "")
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateP2PKH(nil, testAccount.Signer.ID, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	utxo := account.NewUtxo()
	utxo.OutputID = bc.Hash{V0: 1}
	utxo.SourceID = bc.Hash{V0: 2}
	utxo.AssetID = *consensus.BTMAssetID
	utxo.Amount = 1000000000
	utxo.SourcePos = 0
	utxo.ControlProgram = controlProg.ControlProgram
	utxo.AccountID = controlProg.AccountID
	utxo.Address = controlProg.Address
	utxo.ControlProgramIndex = controlProg.KeyIndex
	txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo, nil)
	if err != nil {
		t.Fatal(err)
	}

	b := txbuilder.NewBuilder(time.Now())
	b.AddInput(txInput, sigInst)
	out := legacy.NewTxOutput(*consensus.BTMAssetID, 100, []byte{byte(vm.OP_FAIL)}, nil)
	b.AddOutput(out)
	tpl, tx, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.Sign(nil, tpl, nil, "password", func(_ context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
		sigBytes, err := hsm.XSign(xpub, path, data[:], password)
		if err != nil {
			return nil, nil
		}
		return sigBytes, err
	})
	if err != nil {
		t.Fatal(err)
	}

	bcBlock := &bc.Block{
		BlockHeader: &bc.BlockHeader{Height: 1},
	}
	if _, err = validation.ValidateTx(legacy.MapTx(tx), bcBlock); err != nil {
		t.Fatal(err)
	}
}

func mockChain(testDB dbm.DB) (*protocol.Chain, error) {
	store := txdb.NewStore(testDB)
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(bc.Hash{}, store, txPool)
	if err != nil {
		return nil, err
	}
	return chain, nil
}
