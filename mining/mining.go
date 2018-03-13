// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm/vmutil"
)

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func createCoinbaseTx(accountManager *account.Manager, amount uint64, blockHeight uint64) (tx *legacy.Tx, err error) {
	amount += consensus.BlockSubsidy(blockHeight)

	var script []byte
	if accountManager == nil {
		script, err = vmutil.DefaultCoinbaseProgram()
	} else {
		script, err = accountManager.GetCoinbaseControlProgram()
	}
	if err != nil {
		return
	}

	builder := txbuilder.NewBuilder(time.Now())
	if err = builder.AddInput(legacy.NewCoinbaseInput([]byte(string(blockHeight))), &txbuilder.SigningInstruction{}); err != nil {
		return
	}
	if err = builder.AddOutput(legacy.NewTxOutput(*consensus.BTMAssetID, amount, script)); err != nil {
		return
	}
	_, txData, err := builder.Build()
	if err != nil {
		return
	}

	tx = &legacy.Tx{
		TxData: *txData,
		Tx:     legacy.MapTx(txData),
	}
	return
}

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(c *protocol.Chain, txPool *protocol.TxPool, accountManager *account.Manager) (b *legacy.Block, err error) {
	view := state.NewUtxoViewpoint()
	txStatus := bc.NewTransactionStatus()
	txEntries := []*bc.Tx{nil}
	gasUsed := uint64(0)
	txFee := uint64(0)

	// get preblock info for generate next block
	preBlock := c.BestBlock()
	preBcBlock := legacy.MapBlock(preBlock)
	nextBlockHeight := preBlock.BlockHeader.Height + 1

	var compareDiffBH *legacy.BlockHeader
	if compareDiffBlock, err := c.GetBlockByHeight(nextBlockHeight - consensus.BlocksPerRetarget); err == nil {
		compareDiffBH = &compareDiffBlock.BlockHeader
	}

	b = &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            nextBlockHeight,
			PreviousBlockHash: preBlock.Hash(),
			Timestamp:         uint64(time.Now().Unix()),
			BlockCommitment:   legacy.BlockCommitment{},
			Bits:              difficulty.CalcNextRequiredDifficulty(&preBlock.BlockHeader, compareDiffBH),
		},
		Transactions: []*legacy.Tx{nil},
	}
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: nextBlockHeight}}

	txs := txPool.GetTransactions()
	sort.Sort(ByTime(txs))
	for _, txDesc := range txs {
		tx := txDesc.Tx.Tx
		gasOnlyTx := false

		if err := c.GetTransactionsUtxo(view, []*bc.Tx{tx}); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			continue
		}

		gasStatus, err := validation.ValidateTx(tx, preBcBlock)
		if err != nil {
			if !gasStatus.GasVaild {
				log.WithField("error", err).Error("mining block generate skip tx due to")
				txPool.RemoveTransaction(&tx.ID)
				continue
			}
			gasOnlyTx = true
		}

		if gasUsed+uint64(gasStatus.GasUsed) > consensus.MaxBlockGas {
			break
		}

		if err := view.ApplyTransaction(bcBlock, tx, gasOnlyTx); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			continue
		}

		txStatus.SetStatus(len(b.Transactions), gasOnlyTx)
		b.Transactions = append(b.Transactions, txDesc.Tx)
		txEntries = append(txEntries, tx)
		gasUsed += uint64(gasStatus.GasUsed)
		txFee += txDesc.Fee

		if gasUsed == consensus.MaxBlockGas {
			break
		}
	}

	// creater coinbase transaction
	b.Transactions[0], err = createCoinbaseTx(accountManager, txFee, nextBlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "fail on createCoinbaseTx")
	}
	txEntries[0] = b.Transactions[0].Tx

	b.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = bc.MerkleRoot(txEntries)
	b.BlockHeader.BlockCommitment.TransactionStatusHash = bc.EntryID(txStatus)
	return b, err
}
