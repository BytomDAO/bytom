package test

import (
	"context"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/blockchain/txdb"
	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm"
)

// Mock transaction pool
func MockTxPool() *protocol.TxPool {
	return protocol.NewTxPool()
}

func MockChain(testDB dbm.DB) (*protocol.Chain, error) {
	store := txdb.NewStore(testDB)
	txPool := MockTxPool()
	genesisBlock := cfg.GenerateGenesisBlock()
	chain, err := protocol.NewChain(genesisBlock.Hash(), store, txPool)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func MockUTXO(controlProg *account.CtrlProgram) *account.UTXO {
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

func MockTx(utxo *account.UTXO, testAccount *account.Account) (*txbuilder.Template, *legacy.TxData, error) {
	txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo)
	if err != nil {
		return nil, nil, err
	}

	b := txbuilder.NewBuilder(time.Now())
	b.AddInput(txInput, sigInst)
	out := legacy.NewTxOutput(*consensus.BTMAssetID, 100, []byte{byte(vm.OP_FAIL)})
	b.AddOutput(out)
	return b.Build()
}

func MockSign(tpl *txbuilder.Template, hsm *pseudohsm.HSM, password string) (bool, error) {
	err := txbuilder.Sign(nil, tpl, nil, password, func(_ context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
		return hsm.XSign(xpub, path, data[:], password)
	})
	if err != nil {
		return false, err
	}
	return txbuilder.SignProgress(tpl), nil
}

// Mock block
func MockBlock() *bc.Block {
	return &bc.Block{
		BlockHeader: &bc.BlockHeader{Height: 1},
	}
}
