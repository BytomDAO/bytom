package test

import (
	"fmt"
	"io/ioutil"
	//"os"
	"testing"
	"time"
	"errors"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/database/storage"
	"github.com/bytom/mining"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/blockchain/signers"
)

func TestInsertChain(t *testing.T) {
	testNumber := 3
	blockTxNumber := 10
	totalTxNumber := testNumber * blockTxNumber
	otherAssetNum := 2

	chain, txs, err := GenerateChainData(totalTxNumber, otherAssetNum, "P2PKH")
	if err != nil {
		t.Fatal("GenerateChainData err:", err)
	}

	for i := 0; i < testNumber; i++ {
		testTxs := txs[blockTxNumber*i : blockTxNumber*(i+1)]
		if err := InsertChain(chain, testTxs); err != nil {
			t.Fatal("Failed to insert block into chain:", err)
		}
	}
}

func GenerateChainData(txNumber, otherAssetNum int, txType string) (*protocol.Chain, []*types.Tx, error) {
	dirPath, err := ioutil.TempDir(".", "testDB")
	if err != nil {
		return nil, nil, err
	}

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	// os.RemoveAll(dirPath)

	// generate transactions
	txs := []*types.Tx{}
	switch txType {
	case "P2PKH":
		txs, err = MockTxsP2PKH(dirPath, testDB, txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, err
		}
	case "P2SH":
		txs, err = MockTxsP2SH(dirPath, testDB, txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, err
		}
	case "MutiSign":
		txs, err = MockTxsMutiSign(dirPath, testDB, txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, err
		}
	default:
		txs, err = CreateTxbyNum(txNumber, otherAssetNum)
		if err != nil {
			return nil, nil, err
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
		return nil, nil, err
	}

	txPool := protocol.NewTxPool()
	store := leveldb.NewStore(testDB)
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		return nil, nil, err
	}

	return chain, txs, nil
}

func InsertChain(chain *protocol.Chain, txs []*types.Tx) error {
	if err := InsertTxPool(chain, txs); err != nil {
		return err
	}

	block, err := CreateBlock(chain)
	if err != nil {
		return err
	}

	blockSize, err := block.MarshalText()
	if err != nil {
		return err
	}

	fmt.Println("------------blocksize:", uint64(len(blockSize)))

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

func InsertTxPool(chain *protocol.Chain, txs []*types.Tx) error {
	for _, tx := range txs {
		if err := txbuilder.FinalizeTx(nil, chain, tx); err != nil {
			return err
		}
	}

	return nil
}

func CreateBlock(chain *protocol.Chain) (b *types.Block, err error) {
	txpool := chain.GetTxPool()
	go processNewTxch(txpool)
	return mining.NewBlockTemplate(chain, txpool, nil)
}

func processNewTxch(txPool *protocol.TxPool) {
	newTxCh := txPool.GetNewTxCh()
	for tx := range newTxCh {
		if tx == nil {}
	}
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

func MockSimpleUtxo(index uint64, assetId *bc.AssetID, amount uint64, ctrlProg *account.CtrlProgram) *account.UTXO {
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
		AssetID:             *assetId,
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
		V0: uint64(18446744073709551615),
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
		txOutput := AddTxOutput(baseUtxo.AssetID, baseUtxo.Amount-uint64(10000000), baseUtxo.ControlProgram)
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

		//fmt.Println("------------------txsize:", tx.Tx.SerializedSize)
		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}

func CreateTxbyNum(txNumber, otherAssetNum int) ([]*types.Tx, error) {
	// generate utxos and transactions
	baseUtxos := GenerateBaseUtxos(txNumber, 624000000000, nil)
	otherUtxos := make([][]*account.UTXO, 0, txNumber)
	if otherAssetNum != 0 {
		for i := 0; i < txNumber; i++ {
			utxos := GenerateOtherUtxos(i, otherAssetNum, 6000,nil)
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
	if err := leveldb.SaveUtxoView(batch, view); err != nil {
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

	xpub, err := hsm.XCreate("TestP2PKH", "password")
	if err != nil {
		return nil, err
	}

	txs := []*types.Tx{}
	for i:= 0; i<txNumber; i++ {
		testAccountAlias := fmt.Sprintf("testAccount%d", i)
		testAccount, err := accountManager.Create(nil, []chainkd.XPub{xpub.XPub}, 1, testAccountAlias, nil)
		if err != nil {
			return nil, err
		}

		controlProg, err := accountManager.CreateAddress(nil, testAccount.ID, false)
		if err != nil {
			return nil, err
		}

		//utxo := MockUTXO(controlProg)
		//tpl, _, err := MockTx(utxo, testAccount)
		utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
		otherUtxos := GenerateOtherUtxos(i, otherAssetNum, 6000, controlProg)
		tpl, err := BuildTx(utxo, otherUtxos, testAccount.Signer)
		if err != nil {
			return nil, err
		}

		if _, err := MockSign(tpl, hsm, "password"); err != nil {
			return nil, err
		}

		//fmt.Println("---------------------tx size:", tpl.Transaction.Tx.SerializedSize)
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

	xpub1, err := hsm.XCreate("TestP2SH1", "password")
	if err != nil {
		return nil, err
	}

	xpub2, err := hsm.XCreate("TestP2SH2", "password")
	if err != nil {
		return nil, err
	}

	txs := []*types.Tx{}
	for i:= 0; i<txNumber; i++ {
		testAccountAlias := fmt.Sprintf("testAccount%d", i)
		testAccount, err := accountManager.Create(nil, []chainkd.XPub{xpub1.XPub, xpub2.XPub}, 2, testAccountAlias, nil)
		if err != nil {
			return nil, err
		}

		controlProg, err := accountManager.CreateAddress(nil, testAccount.ID, false)
		if err != nil {
			return nil, err
		}

		//utxo := MockUTXO(controlProg)
		//tpl, _, err := MockTx(utxo, testAccount)
		utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
		otherUtxos := GenerateOtherUtxos(i, otherAssetNum, 6000, controlProg)
		tpl, err := BuildTx(utxo, otherUtxos, testAccount.Signer)
		if err != nil {
			return nil, err
		}

		if _, err := MockSign(tpl, hsm, "password"); err != nil {
			return nil, err
		}

		//fmt.Println("---------------------tx size:", tpl.Transaction.Tx.SerializedSize)
		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}

func MockTxsMutiSign(keyDirPath string, testDB dbm.DB, txNumber, otherAssetNum int) ([]*types.Tx, error) {
	accountManager := account.NewManager(testDB, nil)
	hsm, err := pseudohsm.New(keyDirPath)
	if err != nil {
		return nil, err
	}

	xpub1, err := hsm.XCreate("TestMutilNodeSign1", "password1")
	if err != nil {
		return nil, err
	}

	xpub2, err := hsm.XCreate("TestMutilNodeSign2", "password2")
	if err != nil {
		return nil, err
	}
	txs := []*types.Tx{}
	for i:= 0; i<txNumber; i++ {
		testAccountAlias := fmt.Sprintf("testAccount%d", i)
		testAccount, err := accountManager.Create(nil, []chainkd.XPub{xpub1.XPub, xpub2.XPub}, 2, testAccountAlias, nil)
		if err != nil {
			return nil, err
		}

		controlProg, err := accountManager.CreateAddress(nil, testAccount.ID, false)
		if err != nil {
			return nil, err
		}

		//utxo := MockUTXO(controlProg)
		//tpl, _, err := MockTx(utxo, testAccount)
		utxo := MockSimpleUtxo(0, consensus.BTMAssetID, 1000000000, controlProg)
		otherUtxos := GenerateOtherUtxos(i, otherAssetNum, 6000, controlProg)
		tpl, err := BuildTx(utxo, otherUtxos, testAccount.Signer)
		if err != nil {
			return nil, err
		}

		if finishSign, err := MockSign(tpl, hsm, "password"); err != nil {
			return nil, err
		} else if finishSign == true {
			err := errors.New("sign progress is finish, but either xpub1 nor xpub2 is signed")
			return nil, err
		}

		if finishSign, err := MockSign(tpl, hsm, "password1"); err != nil {
			return nil, err
		} else if finishSign == true {
			err := errors.New("sign progress is finish, but xpub2 is not signed")
			return nil, err
		}

		if finishSign, err := MockSign(tpl, hsm, "password2"); err != nil {
			return nil, err
		} else if finishSign == false {
			err := errors.New("sign progress is not finish,  but both xpub1 and xpub2 is signed")
			return nil, err
		}

		//fmt.Println("---------------------tx size:", tpl.Transaction.Tx.SerializedSize)
		txs = append(txs, tpl.Transaction)
	}

	return txs, nil
}