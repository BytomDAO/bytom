package test

import (
	"fmt"
	"os"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/golang/protobuf/proto"
)

const utxoPrefix = "UT:"

type chainTestContext struct {
	Chain *protocol.Chain
	DB    dbm.DB
	Store *leveldb.Store
}

func (ctx *chainTestContext) validateStatus(block *types.Block) error {
	// validate in mainchain
	if !ctx.Chain.InMainChain(block.Hash()) {
		return fmt.Errorf("block %d is not in mainchain", block.Height)
	}

	// validate chain status and saved block
	bestBlockHeader := ctx.Chain.BestBlockHeader()
	chainBlock, err := ctx.Chain.GetBlockByHeight(block.Height)
	if err != nil {
		return err
	}

	blockHash := block.Hash()
	if bestBlockHeader.Hash() != blockHash || chainBlock.Hash() != blockHash {
		return fmt.Errorf("chain status error")
	}

	// validate tx status
	txStatus, err := ctx.Chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	txStatusMerkleRoot, err := bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		return err
	}

	if txStatusMerkleRoot != block.TransactionStatusHash {
		return fmt.Errorf("tx status error")
	}
	return nil
}

func (ctx *chainTestContext) validateExecution(block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, spentOutputID := range tx.SpentOutputIDs {
			utxoEntry, _ := ctx.Store.GetUtxo(&spentOutputID)
			if utxoEntry == nil {
				continue
			}
			if !utxoEntry.IsCoinBase {
				return fmt.Errorf("found non-coinbase spent utxo entry")
			}
			if !utxoEntry.Spent {
				return fmt.Errorf("utxo entry status should be spent")
			}
		}

		for _, outputID := range tx.ResultIds {
			utxoEntry, _ := ctx.Store.GetUtxo(outputID)
			if utxoEntry == nil && isSpent(outputID, block) {
				continue
			}
			if utxoEntry.BlockHeight != block.Height {
				return fmt.Errorf("block height error, expected: %d, have: %d", block.Height, utxoEntry.BlockHeight)
			}
			if utxoEntry.Spent {
				return fmt.Errorf("utxo entry status should not be spent")
			}
		}
	}
	return nil
}

func (ctx *chainTestContext) getUtxoEntries() map[string]*storage.UtxoEntry {
	utxoEntries := make(map[string]*storage.UtxoEntry)
	iter := ctx.DB.IteratorPrefix([]byte(utxoPrefix))
	defer iter.Release()

	for iter.Next() {
		utxoEntry := storage.UtxoEntry{}
		if err := proto.Unmarshal(iter.Value(), &utxoEntry); err != nil {
			return nil
		}
		key := string(iter.Key())
		utxoEntries[key] = &utxoEntry
	}
	return utxoEntries
}

func (ctx *chainTestContext) validateRollback(utxoEntries map[string]*storage.UtxoEntry) error {
	newUtxoEntries := ctx.getUtxoEntries()
	for key := range utxoEntries {
		entry, ok := newUtxoEntries[key]
		if !ok {
			return fmt.Errorf("can't find utxo after rollback")
		}
		if entry.Spent != utxoEntries[key].Spent {
			return fmt.Errorf("utxo status dismatch after rollback")
		}
	}
	return nil
}

type chainTestConfig struct {
	RollbackTo uint64     `json:"rollback_to"`
	Blocks     []*ctBlock `json:"blocks"`
}

type ctBlock struct {
	Transactions []*ctTransaction `json:"transactions"`
	Append       uint64           `json:"append"`
	Invalid      bool             `json:"invalid"`
}

func (b *ctBlock) createBlock(ctx *chainTestContext) (*types.Block, error) {
	txs := make([]*types.Tx, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		tx, err := t.createTransaction(ctx, txs)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return NewBlock(ctx.Chain, txs, []byte{byte(vm.OP_TRUE)})
}

type ctTransaction struct {
	Inputs  []*ctInput `json:"inputs"`
	Outputs []uint64   `json:"outputs"`
}

type ctInput struct {
	Height      uint64 `json:"height"`
	TxIndex     uint64 `json:"tx_index"`
	OutputIndex uint64 `json:"output_index"`
}

func (input *ctInput) createTxInput(ctx *chainTestContext) (*types.TxInput, error) {
	block, err := ctx.Chain.GetBlockByHeight(input.Height)
	if err != nil {
		return nil, err
	}

	spendInput, err := CreateSpendInput(block.Transactions[input.TxIndex], input.OutputIndex)
	if err != nil {
		return nil, err
	}

	return &types.TxInput{
		AssetVersion: assetVersion,
		TypedInput:   spendInput,
	}, nil
}

// create tx input spent previous tx output in the same block
func (input *ctInput) createDependencyTxInput(txs []*types.Tx) (*types.TxInput, error) {
	// sub 1 because of coinbase tx is not included in txs
	spendInput, err := CreateSpendInput(txs[input.TxIndex-1], input.OutputIndex)
	if err != nil {
		return nil, err
	}

	return &types.TxInput{
		AssetVersion: assetVersion,
		TypedInput:   spendInput,
	}, nil
}

func (t *ctTransaction) createTransaction(ctx *chainTestContext, txs []*types.Tx) (*types.Tx, error) {
	builder := txbuilder.NewBuilder(time.Now())
	sigInst := &txbuilder.SigningInstruction{}
	currentHeight := ctx.Chain.BestBlockHeight()
	for _, input := range t.Inputs {
		var txInput *types.TxInput
		var err error
		if input.Height == currentHeight+1 {
			txInput, err = input.createDependencyTxInput(txs)
		} else {
			txInput, err = input.createTxInput(ctx)
		}
		if err != nil {
			return nil, err
		}
		builder.AddInput(txInput, sigInst)
	}

	for _, amount := range t.Outputs {
		output := types.NewTxOutput(*consensus.BTMAssetID, amount, []byte{byte(vm.OP_TRUE)})
		builder.AddOutput(output)
	}

	tpl, _, err := builder.Build()
	if err != nil {
		return nil, err
	}

	txSerialized, err := tpl.Transaction.MarshalText()
	if err != nil {
		return nil, err
	}

	tpl.Transaction.Tx.SerializedSize = uint64(len(txSerialized))
	tpl.Transaction.TxData.SerializedSize = uint64(len(txSerialized))
	return tpl.Transaction, err
}

func (cfg *chainTestConfig) Run() error {
	db := dbm.NewDB("chain_test_db", "leveldb", "chain_test_db")
	defer os.RemoveAll("chain_test_db")
	chain, store, _, _ := MockChain(db)
	ctx := &chainTestContext{
		Chain: chain,
		DB:    db,
		Store: store,
	}

	var utxoEntries map[string]*storage.UtxoEntry
	var rollbackBlock *types.Block
	for _, blk := range cfg.Blocks {
		block, err := blk.createBlock(ctx)
		if err != nil {
			return err
		}
		err = SolveAndUpdate(ctx.Chain, block)
		if err != nil && blk.Invalid {
			continue
		}
		if err != nil {
			return err
		}
		if err := ctx.validateStatus(block); err != nil {
			return err
		}
		if err := ctx.validateExecution(block); err != nil {
			return err
		}
		if block.Height <= cfg.RollbackTo && cfg.RollbackTo <= block.Height+blk.Append {
			utxoEntries = ctx.getUtxoEntries()
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
	forkedChain, err := declChain("forked_chain", ctx.Chain, rollbackBlock.Height, ctx.Chain.BestBlockHeight()+1)
	defer os.RemoveAll("forked_chain")
	if err != nil {
		return err
	}

	if err := merge(forkedChain, ctx.Chain); err != nil {
		return err
	}
	return ctx.validateRollback(utxoEntries)
}

// if the output(hash) was spent in block
func isSpent(hash *bc.Hash, block *types.Block) bool {
	for _, tx := range block.Transactions {
		for _, spendOutputID := range tx.SpentOutputIDs {
			if spendOutputID == *hash {
				return true
			}
		}
	}
	return false
}
