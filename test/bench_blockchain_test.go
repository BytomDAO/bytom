package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/blockchain/pseudohsm"
	"github.com/bytom/bytom/blockchain/signers"
	"github.com/bytom/bytom/blockchain/txbuilder"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/consensus/difficulty"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/database"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/mining"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	dbm "github.com/bytom/bytom/database/leveldb"
)

func BenchmarkChain_CoinBaseTx_NoAsset(b *testing.B) {
	benchInsertChain(b, 0, 0, "")
}

func BenchmarkChain_BtmTx_NoAsset_BASE(b *testing.B) {
	benchInsertChain(b, 1, 0, "")
}

func BenchmarkChain_5000BtmTx_NoAsset_BASE(b *testing.B) {
	benchInsertChain(b, 5000, 0, "")
}

func BenchmarkChain_5000BtmTx_1Asset_BASE(b *testing.B) {
	benchInsertChain(b, 5000, 1, "")
}

// standard Transaction
func BenchmarkChain_BtmTx_NoAsset_P2PKH(b *testing.B) {
	benchInsertChain(b, 1000, 0, "P2PKH")
}

func BenchmarkChain_BtmTx_1Asset_P2PKH(b *testing.B) {
	benchInsertChain(b, 1000, 1, "P2PKH")
}

func BenchmarkChain_BtmTx_NoAsset_P2SH(b *testing.B) {
	benchInsertChain(b, 100, 0, "P2SH")
}

func BenchmarkChain_BtmTx_1Asset_P2SH(b *testing.B) {
	benchInsertChain(b, 100, 1, "P2SH")
}

func BenchmarkChain_BtmTx_NoAsset_MultiSign(b *testing.B) {
	benchInsertChain(b, 100, 0, "MultiSign")
}

func BenchmarkChain_BtmTx_1Asset_MultiSign(b *testing.B) {
	benchInsertChain(b, 100, 1, "MultiSign")
}

func benchInsertChain(b *testing.B, blockTxNumber int, otherAssetNum int, txType string) {
	b.StopTimer()
	testNumber := b.N
	totalTxNumber := testNumber * blockTxNumber

	dirPath, err := ioutil.TempDir(".", "testDB")
	if err != nil {
		b.Fatal("create dirPath err:", err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	defer testDB.Close()

	// Generate a chain test data.
	chain, txs, txPool, err := GenerateChainData(dirPath, testDB, totalTxNumber, otherAssetNum, txType)
	if err != nil {
		b.Fatal("GenerateChainData err:", err)
	}

	b.ReportAllocs()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		testTxs := txs[blockTxNumber*i : blockTxNumber*(i+1)]
		if err := InsertChain(chain, txPool, testTxs); err != nil {
			b.Fatal("Failed to insert block into chain:", err)
		}
	}
}

func GenerateChainData(dirPath string, testDB dbm.DB, txNumber, otherAssetNum int, txType string) (*protocol.Chain, []*types.Tx, *protocol.TxPool, error) {
	var err error

	// generate transactions
	txs := []*types.Tx{}
	switch txType {
	case "P2PKH":
		txs, err = MockTxsP2PKH(dirPath, testDB, txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, nil, err
		}
	case "P2SH":
		txs, err = MockTxsP2SH(dirPath, testDB, txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, nil, err
		}
	case "MultiSign":
		txs, err = MockTxsMultiSign(dirPath, testDB, txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, nil, err
		}
	default:
		txs, err = CreateTxbyNum(txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// init UtxoViewpoint
	utxoView := state.NewUtxoViewpoint()
	utxoEntry := storage.NewUtxoEntry(false, 1, false)
	for _, tx := range txs {
		for _, id := range tx.SpentOutputIDs {
			utxoView.Entries[id] = utxoEntry
		}
	}

	if err := SetUtxoView(testDB, utxoView); err != nil {
		return nil, nil, nil, err
	}

	store := database.NewStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		return nil, nil, nil, err
	}

	go processNewTxch(txPool)

	return chain, txs, txPool, nil
}

func InsertChain(chain *protocol.Chain, txPool *protocol.TxPool, txs []*types.Tx) error {
	for _, tx := range txs {
		if err := txbuilder.FinalizeTx(nil, chain, tx); err != nil {
			return err
		}
	}

	block, err := mining.NewBlockTemplate(chain, txPool, nil)
	if err != nil {
		return err
	}

	blockSize, err := block.MarshalText()
	if err != nil {
		return err
	}

	fmt.Println("blocksize:", uint64(len(blockSize)))
	fmt.Println("block tx count:", uint64(len(block.Transactions)))
	fmt.Println("coinbase txsize:", uint64(block.Transactions[0].SerializedSize))
	if len(block.Transactions) > 1 {
		fmt.Println("txsize:", uint64(block.Transactions[1].SerializedSize))
	}

	seed, err := chain.CalcNextSeed(&block.PreviousBlockHash)
	if err != nil {
		return err
	}

	if err := SolveBlock(seed, block); err != nil {
		return err
	}

	if _, err := chain.ProcessBlock(block); err != nil {
		return err
	}

	return nil
}

func processNewTxch(txPool *protocol.TxPool) {
}

func SolveBlock(seed *bc.Hash, block *types.Block) error {
	maxNonce := ^uint64(0) // 2^64 - 1
	header := &block.BlockHeader
	for i := uint64(0); i < maxNonce; i++ {
		header.Nonce = i
		headerHash := header.Hash()
		if difficulty.CheckProofOfWork(&headerHash, seed, header.Bits) {
			return nil
		}
	}
	return nil
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

func GenerateBaseUtxos(num int, amount uint64, ctrlProg *account.CtrlProgram) []*account.UTXO {
	utxos := []*account.UTXO{}
	for i := 0; i < num; i++ {
		utxo := MockSimpleUtxo(uint64(i), consensus.BTMAssetID, amount, ctrlProg)
		utxos = append(utxos, utxo)
	}

	return utxos
}

func GenerateOtherUtxos(typeCount, num int, amount uint64, ctrlProg *account.CtrlProgram) []*account.UTXO {
	utxos := []*account.UTXO{}

	assetID := &bc.AssetID{
		V0: uint64(typeCount),
		V1: uint64(1),
		V2: uint64(0),
		V3: uint64(1),
	}

	for i := 0; i < num; i++ {
		utxo := MockSimpleUtxo(uint64(typeCount*num+i), assetID, amount, ctrlProg)
		utxos = append(utxos, utxo)
	}

	return utxos
}

func AddTxInputFromUtxo(utxo *account.UTXO, singer *signers.Signer) (*types.TxInput, *txbuilder.SigningInstruction, error) {
	txInput, signInst, err := account.UtxoToInputs(singer, utxo)
	if err != nil {
		return nil, nil, err
	}

	return txInput, signInst, nil
}

func AddTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *types.TxOutput {
	out := types.NewTxOutput(assetID, amount, controlProgram)
	return out
}

func CreateTxBuilder(baseUtxo *account.UTXO, btmServiceFlag bool, signer *signers.Signer) (*txbuilder.TemplateBuilder, error) {
	tplBuilder := txbuilder.NewBuilder(time.Now())

	// add input
	txInput, signInst, err := AddTxInputFromUtxo(baseUtxo, signer)
	if err != nil {
		return nil, err
	}
	tplBuilder.AddInput(txInput, signInst)

	// if the btm is the service charge, didn't need to add the output
	if btmServiceFlag {
		txOutput := AddTxOutput(baseUtxo.AssetID, 100, baseUtxo.ControlProgram)
		tplBuilder.AddOutput(txOutput)
	}

	return tplBuilder, nil
}

func AddTxBuilder(tplBuilder *txbuilder.TemplateBuilder, utxo *account.UTXO, signer *signers.Signer) error {
	txInput, signInst, err := AddTxInputFromUtxo(utxo, signer)
	if err != nil {
		return err
	}
	tplBuilder.AddInput(txInput, signInst)

	txOutput := AddTxOutput(utxo.AssetID, utxo.Amount, utxo.ControlProgram)
	tplBuilder.AddOutput(txOutput)

	return nil
}

func BuildTx(baseUtxo *account.UTXO, otherUtxos []*account.UTXO, signer *signers.Signer) (*txbuilder.Template, error) {
	btmServiceFlag := false
	if otherUtxos == nil || len(otherUtxos) == 0 {
		btmServiceFlag = true
	}

	tplBuilder, err := CreateTxBuilder(baseUtxo, btmServiceFlag, signer)
	if err != nil {
		return nil, err
	}

	for _, u := range otherUtxos {
		if err := AddTxBuilder(tplBuilder, u, signer); err != nil {
			return nil, err
		}
	}

	tpl, _, err := tplBuilder.Build()
	if err != nil {
		return nil, err
	}

	return tpl, nil
}

func GenetrateTxbyUtxo(baseUtxo []*account.UTXO, otherUtxo [][]*account.UTXO) ([]*types.Tx, error) {
	tmpUtxo := []*account.UTXO{}
	txs := []*types.Tx{}
	otherUtxoFlag := true

	if len(otherUtxo) == 0 || len(otherUtxo) != len(baseUtxo) {
		otherUtxoFlag = false
	}

	for i := 0; i < len(baseUtxo); i++ {
		if otherUtxoFlag {
			tmpUtxo = otherUtxo[i]
		} else {
			tmpUtxo = nil
		}

		tpl, err := BuildTx(baseUtxo[i], tmpUtxo, nil)
		if err != nil {
			return nil, err
		}

		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}

func CreateTxbyNum(txNumber, otherAssetNum int) ([]*types.Tx, error) {
	baseUtxos := GenerateBaseUtxos(txNumber, 1000000000, nil)
	otherUtxos := make([][]*account.UTXO, 0, txNumber)
	if otherAssetNum != 0 {
		for i := 0; i < txNumber; i++ {
			utxos := GenerateOtherUtxos(i, otherAssetNum, 6000, nil)
			otherUtxos = append(otherUtxos, utxos)
		}
	}

	txs, err := GenetrateTxbyUtxo(baseUtxos, otherUtxos)
	if err != nil {
		return nil, err
	}

	return txs, nil
}

func SetUtxoView(db dbm.DB, view *state.UtxoViewpoint) error {
	batch := db.NewBatch()
	if err := database.SaveUtxoView(batch, view); err != nil {
		return err
	}
	batch.Write()
	return nil
}

//-------------------------Mock actual transaction----------------------------------
func MockTxsP2PKH(keyDirPath string, testDB dbm.DB, txNumber, otherAssetNum int) ([]*types.Tx, error) {
	accountManager := account.NewManager(testDB, nil)
	hsm, err := pseudohsm.New(keyDirPath)
	if err != nil {
		return nil, err
	}

	xpub, _, err := hsm.XCreate("TestP2PKH", "password", "en")
	if err != nil {
		return nil, err
	}

	txs := []*types.Tx{}
	for i := 0; i < txNumber; i++ {
		testAccountAlias := fmt.Sprintf("testAccount%d", i)
		testAccount, err := accountManager.Create([]chainkd.XPub{xpub.XPub}, 1, testAccountAlias, signers.BIP0044)
		if err != nil {
			return nil, err
		}

		controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
		if err != nil {
			return nil, err
		}

		utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
		otherUtxos := GenerateOtherUtxos(i, otherAssetNum, 6000, controlProg)
		tpl, err := BuildTx(utxo, otherUtxos, testAccount.Signer)
		if err != nil {
			return nil, err
		}

		if _, err := MockSign(tpl, hsm, "password"); err != nil {
			return nil, err
		}

		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}

func MockTxsP2SH(keyDirPath string, testDB dbm.DB, txNumber, otherAssetNum int) ([]*types.Tx, error) {
	accountManager := account.NewManager(testDB, nil)
	hsm, err := pseudohsm.New(keyDirPath)
	if err != nil {
		return nil, err
	}

	xpub1, _, err := hsm.XCreate("TestP2SH1", "password", "en")
	if err != nil {
		return nil, err
	}

	xpub2, _, err := hsm.XCreate("TestP2SH2", "password", "en")
	if err != nil {
		return nil, err
	}

	txs := []*types.Tx{}
	for i := 0; i < txNumber; i++ {
		testAccountAlias := fmt.Sprintf("testAccount%d", i)
		testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub, xpub2.XPub}, 2, testAccountAlias, signers.BIP0044)
		if err != nil {
			return nil, err
		}

		controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
		if err != nil {
			return nil, err
		}

		utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
		otherUtxos := GenerateOtherUtxos(i, otherAssetNum, 6000, controlProg)
		tpl, err := BuildTx(utxo, otherUtxos, testAccount.Signer)
		if err != nil {
			return nil, err
		}

		if _, err := MockSign(tpl, hsm, "password"); err != nil {
			return nil, err
		}

		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}

func MockTxsMultiSign(keyDirPath string, testDB dbm.DB, txNumber, otherAssetNum int) ([]*types.Tx, error) {
	accountManager := account.NewManager(testDB, nil)
	hsm, err := pseudohsm.New(keyDirPath)
	if err != nil {
		return nil, err
	}

	xpub1, _, err := hsm.XCreate("TestMultilNodeSign1", "password1", "en")
	if err != nil {
		return nil, err
	}

	xpub2, _, err := hsm.XCreate("TestMultilNodeSign2", "password2", "en")
	if err != nil {
		return nil, err
	}
	txs := []*types.Tx{}
	for i := 0; i < txNumber; i++ {
		testAccountAlias := fmt.Sprintf("testAccount%d", i)
		testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub, xpub2.XPub}, 2, testAccountAlias, signers.BIP0044)
		if err != nil {
			return nil, err
		}

		controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
		if err != nil {
			return nil, err
		}

		utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
		otherUtxos := GenerateOtherUtxos(i, otherAssetNum, 6000, controlProg)
		tpl, err := BuildTx(utxo, otherUtxos, testAccount.Signer)
		if err != nil {
			return nil, err
		}

		if _, err := MockSign(tpl, hsm, "password1"); err != nil {
			return nil, err
		}

		if _, err := MockSign(tpl, hsm, "password2"); err != nil {
			return nil, err
		}

		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}
