// +build functional

package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm"
)

type TxTestConfig struct {
	Keys         []*keyInfo       `json:"keys"`
	Accounts     []*accountInfo   `json:"accounts"`
	Transactions []*ttTransaction `json:"transactions"`
}

func (cfg *TxTestConfig) Run() error {
	dirPath, err := ioutil.TempDir(".", "pseudo_hsm")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		return err
	}

	chainDB := dbm.NewDB("chain_db", "leveldb", "chain_db")
	defer os.RemoveAll("chain_db")
	chain, _, _, _ := MockChain(chainDB)
	txTestDB := dbm.NewDB("tx_test_db", "leveldb", "tx_test_db")
	defer os.RemoveAll("tx_test_db")
	accountManager := account.NewManager(txTestDB, chain)
	assets := asset.NewRegistry(txTestDB, chain)

	generator := NewTxGenerator(accountManager, assets, hsm)
	for _, key := range cfg.Keys {
		if err := generator.createKey(key.Name, key.Password); err != nil {
			return err
		}
	}

	for _, acc := range cfg.Accounts {
		if err := generator.createAccount(acc.Name, acc.Keys, acc.Quorum); err != nil {
			return err
		}
	}

	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:  1,
			Version: 1,
		},
	}
	for _, t := range cfg.Transactions {
		tx, err := t.create(generator)
		if err != nil {
			return err
		}

		tx.TxData.Version = t.Version
		tx.Tx = types.MapTx(&tx.TxData)
		status, err := validation.ValidateTx(tx.Tx, block)
		result := err == nil
		if result != t.Valid {
			return fmt.Errorf("tx %s validate failed, expected: %t, have: %t", t.Describe, t.Valid, result)
		}
		if status == nil {
			continue
		}

		gasOnlyTx := false
		if err != nil && status.GasVaild {
			gasOnlyTx = true
		}
		if gasOnlyTx != t.GasOnly {
			return fmt.Errorf("gas only tx %s validate failed", t.Describe)
		}
		if result && t.TxFee != status.BTMValue {
			return fmt.Errorf("gas used dismatch, expected: %d, have: %d", t.TxFee, status.BTMValue)
		}
	}
	return nil
}

type ttTransaction struct {
	wtTransaction
	Describe string `json:"describe"`
	Version  uint64 `json:"version"`
	Valid    bool   `json:"valid"`
	GasOnly  bool   `json:"gas_only"`
	TxFee    uint64 `json:"tx_fee"`
}

// UnmarshalJSON unmarshal transaction with default version 1
func (t *ttTransaction) UnmarshalJSON(data []byte) error {
	type typeAlias ttTransaction
	tx := &typeAlias{
		Version: 1,
	}

	err := json.Unmarshal(data, tx)
	if err != nil {
		return err
	}
	*t = ttTransaction(*tx)
	return nil
}

func (t *ttTransaction) create(g *TxGenerator) (*types.Tx, error) {
	g.Reset()
	for _, input := range t.Inputs {
		switch input.Type {
		case "spend_account":
			utxo, err := g.mockUtxo(input.AccountAlias, input.AssetAlias, input.Amount)
			if err != nil {
				return nil, err
			}
			if err := g.AddTxInputFromUtxo(utxo, input.AccountAlias); err != nil {
				return nil, err
			}
		case "issue":
			_, err := g.createAsset(input.AccountAlias, input.AssetAlias)
			if err != nil {
				return nil, err
			}
			if err := g.AddIssuanceInput(input.AssetAlias, input.Amount); err != nil {
				return nil, err
			}
		}
	}

	for _, output := range t.Outputs {
		switch output.Type {
		case "output":
			if err := g.AddTxOutput(output.AccountAlias, output.AssetAlias, output.Amount); err != nil {
				return nil, err
			}
		case "retire":
			if err := g.AddRetirement(output.AssetAlias, output.Amount); err != nil {
				return nil, err
			}
		}
	}
	return g.Sign(t.Passwords)
}

func TestTx(t *testing.T) {
	walk(t, txTestDir, func(t *testing.T, name string, test *TxTestConfig) {
		if err := test.Run(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestCoinbaseMature(t *testing.T) {
	db := dbm.NewDB("test_coinbase_mature_db", "leveldb", "test_coinbase_mature_db")
	defer os.RemoveAll("test_coinbase_mature_db")
	chain, _, _, _ := MockChain(db)

	defaultCtrlProg := []byte{byte(vm.OP_TRUE)}
	height := chain.BestBlockHeight()
	block, err := chain.GetBlockByHeight(height)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := CreateTxFromTx(block.Transactions[0], 0, 1000000000, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	txs := []*types.Tx{tx}
	matureHeight := chain.BestBlockHeight() + consensus.CoinbasePendingBlockNumber
	currentHeight := chain.BestBlockHeight()
	for h := currentHeight + 1; h < matureHeight; h++ {
		block, err := NewBlock(chain, txs, defaultCtrlProg)
		if err != nil {
			t.Fatal(err)
		}
		if err := SolveAndUpdate(chain, block); err == nil {
			t.Fatal("spent immature coinbase output success")
		}
		block, err = NewBlock(chain, nil, defaultCtrlProg)
		if err != nil {
			t.Fatal(err)
		}
		if err := SolveAndUpdate(chain, block); err != nil {
			t.Fatal(err)
		}
	}

	block, err = NewBlock(chain, txs, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}
	if err := SolveAndUpdate(chain, block); err != nil {
		t.Fatalf("spent mature coinbase output failed: %s", err)
	}
}

func TestCoinbaseTx(t *testing.T) {
	db := dbm.NewDB("test_coinbase_tx_db", "leveldb", "test_coinbase_tx_db")
	defer os.RemoveAll("test_coinbase_tx_db")
	chain, _, _, _ := MockChain(db)

	defaultCtrlProg := []byte{byte(vm.OP_TRUE)}
	if err := AppendBlocks(chain, 1); err != nil {
		t.Fatal(err)
	}

	block, err := chain.GetBlockByHeight(chain.BestBlockHeight())
	if err != nil {
		t.Fatal(err)
	}

	tx, err := CreateTxFromTx(block.Transactions[0], 0, 1000000000, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	block, err = NewBlock(chain, []*types.Tx{tx}, defaultCtrlProg)
	if err != nil {
		t.Fatal(err)
	}

	coinbaseTx, err := CreateCoinbaseTx(defaultCtrlProg, block.Height, 100000)
	if err != nil {
		t.Fatal(err)
	}

	if err := ReplaceCoinbase(block, coinbaseTx); err != nil {
		t.Fatal(err)
	}

	err = SolveAndUpdate(chain, block)
	if err == nil {
		t.Fatalf("invalid coinbase tx validate success")
	}
}
