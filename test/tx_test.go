package test

import (
	"io/ioutil"
	"os"
	"testing"
	"fmt"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/validation"
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
	chain, _ := MockChain(chainDB)
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

	for _, t := range cfg.Transactions {
		tx, err := t.create(generator)
		if err != nil {
			return err
		}

		status, err := validation.ValidateTx(tx.Tx, MockBlock())
		result := err == nil
		if result != t.Valid {
			return fmt.Errorf("tx %s validate failed: %s", t.Describe, err.Error())
		}

		if result && t.GasUsed != status.BTMValue {
			return fmt.Errorf("gas used dismatch, expected: %d, have: %d", t.GasUsed, gasUsed)
		}
	}
	return nil
}

type ttTransaction struct {
	wtTransaction
	Describe string `json:"describe"`
	Valid    bool   `json:"valid"`
	GasUsed  uint64 `json:"gas_used"`
}

func (t *ttTransaction) create(g *TxGenerator) (*types.Tx, error) {
	g.reset()
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
