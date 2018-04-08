package test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	w "github.com/bytom/wallet"
	dbm "github.com/tendermint/tmlibs/db"
)

const (
	vmVersion       = 1
	blkVersion      = 1
	assetVersion    = 1
	defaultDuration = 10
	maxNonce        = ^uint64(0)
)

type BlockchainTestConfig struct {
	Keys     []*keyInfo     `json:"keys"`
	Accounts []*accountInfo `json:"accounts"`
	Blocks   []*block       `json:"blocks"`
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

type block struct {
	CoinbaseAccount string            `json:"coinbase_account"`
	Transactions    []*transaction    `json:"transactions"`
	PostStates      []*accountBalance `json:"post_states"`
	Append          uint64            `json:"append"`
}

func (b *block) create(ctx *BlockchainTestContext) (*types.Block, error) {
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

func (b *block) verifyPostStates(ctx *BlockchainTestContext) error {
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

type transaction struct {
	Passwords []string  `json:"passwords"`
	Inputs    []*action `json:"inputs"`
	Outputs   []*action `json:"outputs"`
}

// create signed transaction
func (t *transaction) create(ctx *BlockchainTestContext) (*types.Tx, error) {
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

type BlockchainTestContext struct {
	Wallet *w.Wallet
	Chain  *protocol.Chain
}

func (ctx *BlockchainTestContext) createControlProgram(accountName string, change bool) (*account.CtrlProgram, error) {
	acc, err := ctx.Wallet.AccountMgr.FindByAlias(nil, accountName)
	if err != nil {
		return nil, err
	}

	return ctx.Wallet.AccountMgr.CreateAddress(nil, acc.ID, change)
}

func (ctx *BlockchainTestContext) getPubkey(keyAlias string) *chainkd.XPub {
	pubKeys := ctx.Wallet.Hsm.ListKeys()
	for i, key := range pubKeys {
		if key.Alias == keyAlias {
			return &pubKeys[i].XPub
		}
	}
	return nil
}

func (ctx *BlockchainTestContext) createAsset(accountAlias string, assetAlias string) (*asset.Asset, error) {
	acc, err := ctx.Wallet.AccountMgr.FindByAlias(nil, accountAlias)
	if err != nil {
		return nil, err
	}
	return ctx.Wallet.AssetReg.Define(acc.XPubs, len(acc.XPubs), nil, assetAlias, nil)
}

func (ctx *BlockchainTestContext) newBlock(txs []*types.Tx, coinbaseAccount string) (*types.Block, error) {
	prevBlock := ctx.Chain.BestBlock()
	height := prevBlock.Height + 1
	timestamp := prevBlock.Timestamp + defaultDuration
	controlProgram, err := ctx.createControlProgram(coinbaseAccount, true)
	if err != nil {
		return nil, err
	}
	return NewBlock(blkVersion, height, timestamp, prevBlock.Bits, prevBlock.Hash(), txs, controlProgram.ControlProgram)
}

func (ctx *BlockchainTestContext) createKey(name string, password string) error {
	_, err := ctx.Wallet.Hsm.XCreate(name, password)
	return err
}

func (ctx *BlockchainTestContext) createAccount(name string, keys []string, quorum int) error {
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

func (ctx *BlockchainTestContext) solve(block *types.Block) error {
	seed, err := ctx.Chain.GetSeed(block.Height, &block.PreviousBlockHash)
	if err != nil {
		return err
	}
	return Solve(seed, block)
}

func (ctx *BlockchainTestContext) updateState(block *types.Block) error {
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

func (ctx *BlockchainTestContext) getBalance(accountAlias string, assetAlias string) (uint64, error) {
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

func (ctx *BlockchainTestContext) append(blkNum uint64) error {
	for i := uint64(0); i < blkNum; i++ {
		prevBlock := ctx.Chain.BestBlock()
		timestamp := prevBlock.Timestamp + 10
		prevBlockHash := prevBlock.Hash()
		block, err := DefaultEmptyBlock(prevBlock.Height+1, timestamp, prevBlockHash, prevBlock.Bits)
		if err != nil {
			return err
		}
		if err := ctx.solve(block); err != nil {
			return err
		}
		if err := ctx.updateState(block); err != nil {
			return nil
		}
	}
	return nil
}

func (config *BlockchainTestConfig) Run() error {
	dirPath, err := ioutil.TempDir(".", "TestP2PKH")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dirPath)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		return err
	}

	txdb := dbm.NewDB("txdb", "leveldb", "txdb")
	defer os.RemoveAll("txdb")
	chain, _ := MockChain(txdb)
	walletDB := dbm.NewDB("wallet", "leveldb", "walletTemp")
	defer os.RemoveAll("walletTemp")
	accountManager := account.NewManager(walletDB, chain)
	assets := asset.NewRegistry(walletDB, chain)
	wallet, err := w.NewWallet(walletDB, accountManager, assets, hsm, chain)
	if err != nil {
		return err
	}
	ctx := &BlockchainTestContext{
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
		if err := ctx.updateState(block); err != nil {
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

func NewBlock(version, height, timestamp, bits uint64, prevBlockHash bc.Hash, txs []*types.Tx, controlProgram []byte) (*types.Block, error) {
	gas := uint64(0)
	transactions := []*types.Tx{nil}
	txEntries := []*bc.Tx{nil}
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	for i, tx := range txs {
		gas += gasUsed(tx)
		transactions = append(transactions, tx)
		// TODO: validate tx
		txEntries = append(txEntries, tx.Tx)
		txStatus.SetStatus(i+1, false)
	}

	coinbaseTx, err := CreateCoinbaseTx(controlProgram, height, gas)
	if err != nil {
		return nil, err
	}
	transactions[0] = coinbaseTx
	txEntries[0] = coinbaseTx.Tx
	txMerkleRoot, err := bc.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}
	txStatusMerkleRoot, err := bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		return nil, err
	}

	b := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           version,
			Height:            height,
			Timestamp:         timestamp,
			Bits:              bits,
			PreviousBlockHash: prevBlockHash,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: txMerkleRoot,
				TransactionStatusHash:  txStatusMerkleRoot,
			},
		},
		Transactions: transactions,
	}
	return b, nil
}

func DefaultEmptyBlock(height uint64, timestamp uint64, prevBlockHash bc.Hash, bits uint64) (*types.Block, error) {
	coinbaseTx, err := DefaultCoinbaseTx(height)
	if err != nil {
		return nil, err
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           blkVersion,
			Height:            height,
			Timestamp:         timestamp,
			PreviousBlockHash: prevBlockHash,
			Bits:              bits,
		},
		Transactions: []*types.Tx{coinbaseTx},
	}
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	block.TransactionsMerkleRoot, err = bc.TxMerkleRoot([]*bc.Tx{coinbaseTx.Tx})
	if err != nil {
		return nil, err
	}

	txStatusMerkleRoot, err := bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	block.TransactionStatusHash = txStatusMerkleRoot
	return block, err
}

func Solve(seed *bc.Hash, block *types.Block) error {
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
