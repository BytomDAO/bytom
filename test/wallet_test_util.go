package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/types"
	w "github.com/bytom/wallet"
)

type walletTestConfig struct {
	Keys       []*keyInfo     `json:"keys"`
	Accounts   []*accountInfo `json:"accounts"`
	Blocks     []*wtBlock     `json:"blocks"`
	RollbackTo uint64         `json:"rollback_to"`
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

type wtBlock struct {
	CoinbaseAccount string            `json:"coinbase_account"`
	Transactions    []*wtTransaction  `json:"transactions"`
	PostStates      []*accountBalance `json:"post_states"`
	Append          uint64            `json:"append"`
}

func (b *wtBlock) create(ctx *walletTestContext) (*types.Block, error) {
	transactions := make([]*types.Tx, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		tx, err := t.create(ctx)
		if err != nil {
			continue
		}
		transactions = append(transactions, tx)
	}
	return ctx.newBlock(transactions, b.CoinbaseAccount)
}

func (b *wtBlock) verifyPostStates(ctx *walletTestContext) error {
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

type wtTransaction struct {
	Passwords []string  `json:"passwords"`
	Inputs    []*action `json:"inputs"`
	Outputs   []*action `json:"outputs"`
}

// create signed transaction
func (t *wtTransaction) create(ctx *walletTestContext) (*types.Tx, error) {
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

type walletTestContext struct {
	Wallet *w.Wallet
	Chain  *protocol.Chain
}

func (ctx *walletTestContext) createControlProgram(accountName string) (*account.CtrlProgram, error) {
	acc, err := ctx.Wallet.AccountMgr.FindByAlias(nil, accountName)
	if err != nil {
		return nil, err
	}

	return ctx.Wallet.AccountMgr.CreateAddress(nil, acc.ID)
}

func (ctx *walletTestContext) getPubkey(keyAlias string) *chainkd.XPub {
	pubKeys := ctx.Wallet.Hsm.ListKeys()
	for i, key := range pubKeys {
		if key.Alias == keyAlias {
			return &pubKeys[i].XPub
		}
	}
	return nil
}

func (ctx *walletTestContext) createAsset(accountAlias string, assetAlias string) (*asset.Asset, error) {
	acc, err := ctx.Wallet.AccountMgr.FindByAlias(nil, accountAlias)
	if err != nil {
		return nil, err
	}
	return ctx.Wallet.AssetReg.Define(acc.XPubs, len(acc.XPubs), nil, assetAlias, nil)
}

func (ctx *walletTestContext) newBlock(txs []*types.Tx, coinbaseAccount string) (*types.Block, error) {
	controlProgram, err := ctx.createControlProgram(coinbaseAccount)
	if err != nil {
		return nil, err
	}
	return NewBlock(ctx.Chain, txs, controlProgram.ControlProgram)
}

func (ctx *walletTestContext) createKey(name string, password string) error {
	_, err := ctx.Wallet.Hsm.XCreate(name, password)
	return err
}

func (ctx *walletTestContext) createAccount(name string, keys []string, quorum int) error {
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

func (ctx *walletTestContext) update(block *types.Block) error {
	if err := SolveAndUpdate(ctx.Chain, block); err != nil {
		return err
	}
	if err := ctx.Wallet.AttachBlock(block); err != nil {
		return err
	}
	return nil
}

func (ctx *walletTestContext) getBalance(accountAlias string, assetAlias string) (uint64, error) {
	balances := ctx.Wallet.GetAccountBalances("")
	for _, balance := range balances {
		if balance.Alias == accountAlias && balance.AssetAlias == assetAlias {
			return balance.Amount, nil
		}
	}
	return 0, nil
}

func (ctx *walletTestContext) getAccBalances() map[string]map[string]uint64 {
	accBalances := make(map[string]map[string]uint64)
	balances := ctx.Wallet.GetAccountBalances("")
	for _, balance := range balances {
		if accBalance, ok := accBalances[balance.Alias]; ok {
			if _, ok := accBalance[balance.AssetAlias]; ok {
				accBalance[balance.AssetAlias] += balance.Amount
				continue
			}
			accBalance[balance.AssetAlias] = balance.Amount
			continue
		}
		accBalances[balance.Alias] = map[string]uint64{balance.AssetAlias: balance.Amount}
	}
	return accBalances
}

func (ctx *walletTestContext) getDetachedBlocks(height uint64) ([]*types.Block, error) {
	currentHeight := ctx.Chain.BestBlockHeight()
	detachedBlocks := make([]*types.Block, 0, currentHeight-height)
	for i := currentHeight; i > height; i-- {
		block, err := ctx.Chain.GetBlockByHeight(i)
		if err != nil {
			return detachedBlocks, err
		}
		detachedBlocks = append(detachedBlocks, block)
	}
	return detachedBlocks, nil
}

func (ctx *walletTestContext) validateRollback(oldAccBalances map[string]map[string]uint64) error {
	accBalances := ctx.getAccBalances()
	if reflect.DeepEqual(oldAccBalances, accBalances) {
		return nil
	}
	return fmt.Errorf("different account balances after rollback")
}

func (cfg *walletTestConfig) Run() error {
	dirPath, err := ioutil.TempDir(".", "pseudo_hsm")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		return err
	}

	db := dbm.NewDB("wallet_test_db", "leveldb", "wallet_test_db")
	defer os.RemoveAll("wallet_test_db")
	chain, _, _, _ := MockChain(db)
	walletDB := dbm.NewDB("wallet", "leveldb", "wallet_db")
	defer os.RemoveAll("wallet_db")
	accountManager := account.NewManager(walletDB, chain)
	assets := asset.NewRegistry(walletDB, chain)
	wallet, err := w.NewWallet(walletDB, accountManager, assets, hsm, chain)
	if err != nil {
		return err
	}
	ctx := &walletTestContext{
		Wallet: wallet,
		Chain:  chain,
	}

	for _, key := range cfg.Keys {
		if err := ctx.createKey(key.Name, key.Password); err != nil {
			return err
		}
	}

	for _, acc := range cfg.Accounts {
		if err := ctx.createAccount(acc.Name, acc.Keys, acc.Quorum); err != nil {
			return err
		}
	}

	var accBalances map[string]map[string]uint64
	var rollbackBlock *types.Block
	for _, blk := range cfg.Blocks {
		block, err := blk.create(ctx)
		if err != nil {
			return err
		}
		if err := ctx.update(block); err != nil {
			return err
		}
		if err := blk.verifyPostStates(ctx); err != nil {
			return err
		}
		if block.Height <= cfg.RollbackTo && cfg.RollbackTo <= block.Height+blk.Append {
			accBalances = ctx.getAccBalances()
			rollbackBlock = block
		}
		if err := AppendBlocks(ctx.Chain, blk.Append); err != nil {
			return err
		}
	}

	if rollbackBlock == nil {
		return nil
	}

	// rollback and validate
	detachedBlocks, err := ctx.getDetachedBlocks(rollbackBlock.Height)
	if err != nil {
		return err
	}

	forkedChain, err := declChain("forked_chain", ctx.Chain, rollbackBlock.Height, ctx.Chain.BestBlockHeight()+1)
	defer os.RemoveAll("forked_chain")
	if err != nil {
		return err
	}

	if err := merge(forkedChain, ctx.Chain); err != nil {
		return err
	}

	for _, block := range detachedBlocks {
		if err := ctx.Wallet.DetachBlock(block); err != nil {
			return err
		}
	}
	return ctx.validateRollback(accBalances)
}
