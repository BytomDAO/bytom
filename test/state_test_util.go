package test

import (
	"fmt"
	"io/ioutil"
	"os"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/types"
	w "github.com/bytom/wallet"
)

type StateTestConfig struct {
	Keys     []*keyInfo     `json:"keys"`
	Accounts []*accountInfo `json:"accounts"`
	Blocks   []*stBlock     `json:"blocks"`
}

type keyInfo struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type accountInfo struct {
	Name   string   `json:"name"`
	Keys   []string `json:"keys"`
	Quorum int      `json:"quorum"`
}

type stBlock struct {
	CoinbaseAccount string            `json:"coinbase_account"`
	Transactions    []*stTransaction  `json:"transactions"`
	PostStates      []*accountBalance `json:"post_states"`
	Append          uint64            `json:"append"`
}

func (b *stBlock) create(ctx *StateTestContext) (*types.Block, error) {
	transactions := []*types.Tx{}
	for _, t := range b.Transactions {
		tx, err := t.create(ctx)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	return ctx.newBlock(transactions, b.CoinbaseAccount)
}

func (b *stBlock) verifyPostStates(ctx *StateTestContext) error {
	for _, state := range b.PostStates {
		balance, err := ctx.getBalance(state.AccountAlias, state.AssetAlias)
		if err != nil {
			return err
		}

		if balance != state.Amount {
			return fmt.Errorf("AccountAlias: %s, AssetAlias: %s, expected: %d, have: %d", state.AccountAlias, state.AssetAlias, state.Amount, balance)
		}
	}
	return nil
}

type stTransaction struct {
	Passwords []string  `json:"passwords"`
	Inputs    []*action `json:"inputs"`
	Outputs   []*action `json:"outputs"`
}

// create signed transaction
func (t *stTransaction) create(ctx *StateTestContext) (*types.Tx, error) {
	generator := NewTxGenerator(ctx.Wallet.AccountMgr, ctx.Wallet.AssetReg, ctx.Wallet.Hsm)
	for _, input := range t.Inputs {
		switch input.Type {
		case "spend_account":
			if err := generator.AddSpendInput(input.AccountAlias, input.AssetAlias, input.Amount); err != nil {
				return nil, err
			}
		case "issue":
			_, err := ctx.createAsset(input.AccountAlias, input.AssetAlias)
			if err != nil {
				return nil, err
			}
			if err := generator.AddIssuanceInput(input.AssetAlias, input.Amount); err != nil {
				return nil, err
			}
		}
	}

	for _, output := range t.Outputs {
		switch output.Type {
		case "output":
			if err := generator.AddTxOutput(output.AccountAlias, output.AssetAlias, output.Amount); err != nil {
				return nil, err
			}
		case "retire":
			if err := generator.AddRetirement(output.AssetAlias, output.Amount); err != nil {
				return nil, err
			}
		}
	}
	return generator.Sign(t.Passwords)
}

type action struct {
	Type         string `json:"type"`
	AccountAlias string `json:"name"`
	AssetAlias   string `json:"asset"`
	Amount       uint64 `json:"amount"`
}

type accountBalance struct {
	AssetAlias   string `json:"asset"`
	AccountAlias string `json:"name"`
	Amount       uint64 `json:"amount"`
}

type StateTestContext struct {
	Wallet *w.Wallet
	Chain  *protocol.Chain
}

func (ctx *StateTestContext) createControlProgram(accountName string, change bool) (*account.CtrlProgram, error) {
	acc, err := ctx.Wallet.AccountMgr.FindByAlias(nil, accountName)
	if err != nil {
		return nil, err
	}

	return ctx.Wallet.AccountMgr.CreateAddress(nil, acc.ID, change)
}

func (ctx *StateTestContext) getPubkey(keyAlias string) *chainkd.XPub {
	pubKeys := ctx.Wallet.Hsm.ListKeys()
	for i, key := range pubKeys {
		if key.Alias == keyAlias {
			return &pubKeys[i].XPub
		}
	}
	return nil
}

func (ctx *StateTestContext) createAsset(accountAlias string, assetAlias string) (*asset.Asset, error) {
	acc, err := ctx.Wallet.AccountMgr.FindByAlias(nil, accountAlias)
	if err != nil {
		return nil, err
	}
	return ctx.Wallet.AssetReg.Define(acc.XPubs, len(acc.XPubs), nil, assetAlias, nil)
}

func (ctx *StateTestContext) newBlock(txs []*types.Tx, coinbaseAccount string) (*types.Block, error) {
	prevBlock := ctx.Chain.BestBlock()
	height := prevBlock.Height + 1
	timestamp := prevBlock.Timestamp + defaultDuration
	controlProgram, err := ctx.createControlProgram(coinbaseAccount, true)
	if err != nil {
		return nil, err
	}
	return NewBlock(blkVersion, height, timestamp, prevBlock.Bits, prevBlock.Hash(), txs, controlProgram.ControlProgram)
}

func (ctx *StateTestContext) createKey(name string, password string) error {
	_, err := ctx.Wallet.Hsm.XCreate(name, password)
	return err
}

func (ctx *StateTestContext) createAccount(name string, keys []string, quorum int) error {
	xpubs := []chainkd.XPub{}
	for _, alias := range keys {
		xpub := ctx.getPubkey(alias)
		if xpub == nil {
			return fmt.Errorf("can't find pubkey for %s", alias)
		}
		xpubs = append(xpubs, *xpub)
	}
	_, err := ctx.Wallet.AccountMgr.Create(nil, xpubs, quorum, name, nil)
	return err
}

func (ctx *StateTestContext) solve(block *types.Block) error {
	seed, err := ctx.Chain.GetSeed(block.Height, &block.PreviousBlockHash)
	if err != nil {
		return err
	}
	return Solve(seed, block)
}

func (ctx *StateTestContext) update(block *types.Block) error {
	if err := ctx.Chain.SaveBlock(block); err != nil {
		return err
	}
	if err := ctx.Chain.ConnectBlock(block); err != nil {
		return err
	}
	if err := ctx.Wallet.AttachBlock(block); err != nil {
		return err
	}
	return nil
}

func (ctx *StateTestContext) getBalance(accountAlias string, assetAlias string) (uint64, error) {
	balances, err := ctx.Wallet.GetAccountBalances("")
	if err != nil {
		return 0, err
	}

	for _, balance := range balances {
		if balance.Alias == accountAlias && balance.AssetAlias == assetAlias {
			return balance.Amount, nil
		}
	}
	return 0, nil
}

func (ctx *StateTestContext) append(blkNum uint64) error {
	for i := uint64(0); i < blkNum; i++ {
		prevBlock := ctx.Chain.BestBlock()
		timestamp := prevBlock.Timestamp + defaultDuration
		prevBlockHash := prevBlock.Hash()
		block, err := DefaultEmptyBlock(prevBlock.Height+1, timestamp, prevBlockHash, prevBlock.Bits)
		if err != nil {
			return err
		}
		if err := ctx.solve(block); err != nil {
			return err
		}
		if err := ctx.update(block); err != nil {
			return nil
		}
	}
	return nil
}

func (config *StateTestConfig) Run() error {
	dirPath, err := ioutil.TempDir(".", "pseudo_hsm")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		return err
	}

	db := dbm.NewDB("state_test_db", "leveldb", "state_test_db")
	defer os.RemoveAll("state_test_db")
	chain, _ := MockChain(db)
	walletDB := dbm.NewDB("wallet", "leveldb", "wallet_db")
	defer os.RemoveAll("wallet_db")
	accountManager := account.NewManager(walletDB, chain)
	assets := asset.NewRegistry(walletDB, chain)
	wallet, err := w.NewWallet(walletDB, accountManager, assets, hsm, chain)
	if err != nil {
		return err
	}
	ctx := &StateTestContext{
		Wallet: wallet,
		Chain:  chain,
	}

	for _, key := range config.Keys {
		if err := ctx.createKey(key.Name, key.Password); err != nil {
			return err
		}
	}

	for _, acc := range config.Accounts {
		if err := ctx.createAccount(acc.Name, acc.Keys, acc.Quorum); err != nil {
			return err
		}
	}

	for _, blk := range config.Blocks {
		block, err := blk.create(ctx)
		if err != nil {
			return err
		}
		if err := ctx.solve(block); err != nil {
			return err
		}
		if err := ctx.update(block); err != nil {
			return err
		}
		if err := blk.verifyPostStates(ctx); err != nil {
			return err
		}
		if err := ctx.append(blk.Append); err != nil {
			return err
		}
	}
	return nil
}
