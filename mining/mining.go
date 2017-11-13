// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
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
	unlockHeight := blockHeight + consensus.CoinbasePendingBlockNumber

	var script []byte
	if accountManager == nil {
		script, err = vmutil.CoinbaseProgram(nil, 0, unlockHeight)
	} else {
		script, err = accountManager.GetCoinbaseControlProgram(unlockHeight)
	}
	if err != nil {
		return
	}

	builder := txbuilder.NewBuilder(time.Now())
	builder.AddOutput(legacy.NewTxOutput(*consensus.BTMAssetID, amount, script, nil))
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
func NewBlockTemplate(c *protocol.Chain, txPool *protocol.TxPool, accountManager *account.Manager) (*legacy.Block, error) {
	// Extend the most recently known best block.
	var err error
	preBlock, snap := c.State()
	newSnap := state.Copy(snap)

	nextBlockHeight := preBlock.BlockHeader.Height + 1
	preBcBlock := legacy.MapBlock(preBlock)
	txDescs := txPool.GetTransactions()
	txEntries := make([]*bc.Tx, 0, len(txDescs))
	blockWeight := uint64(0)
	txFee := uint64(0)

	var compareDiffBH *legacy.BlockHeader
	if compareDiffBlock, err := c.GetBlockByHeight(nextBlockHeight - consensus.BlocksPerRetarget); err == nil {
		compareDiffBH = &compareDiffBlock.BlockHeader
	}

	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            nextBlockHeight,
			PreviousBlockHash: preBlock.Hash(),
			TimestampMS:       bc.Millis(time.Now()),
			BlockCommitment:   legacy.BlockCommitment{},
			Bits:              consensus.CalcNextRequiredDifficulty(&preBlock.BlockHeader, compareDiffBH),
		},
		Transactions: make([]*legacy.Tx, 0, len(txDescs)),
	}
	newSnap.PruneNonces(b.BlockHeader.TimestampMS)

	appendTx := func(tx *legacy.Tx, weight, fee uint64) {
		b.Transactions = append([]*legacy.Tx{tx}, b.Transactions...)
		txEntries = append([]*bc.Tx{tx.Tx}, txEntries...)
		blockWeight += weight
		txFee += fee
	}

	for _, txDesc := range txDescs {
		tx := txDesc.Tx.Tx
		if blockWeight+txDesc.Weight > consensus.MaxBlockSzie-consensus.MaxTxSize {
			break
		}
		if err := newSnap.ApplyTx(tx); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			continue
		}
		if _, err := validation.ValidateTx(tx, preBcBlock); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			continue
		}

		appendTx(txDesc.Tx, txDesc.Weight, txDesc.Fee)
	}

	cbTx, err := createCoinbaseTx(accountManager, txFee, nextBlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "fail on createCoinbaseTx")
	}
	if err := newSnap.ApplyTx(cbTx.Tx); err != nil {
		return nil, errors.Wrap(err, "fail on append coinbase transaction to snap")
	}
	appendTx(cbTx, 0, 0)

	b.BlockHeader.BlockCommitment.AssetsMerkleRoot = newSnap.Tree.RootHash()
	b.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = bc.MerkleRoot(txEntries)
	if err != nil {
		return nil, errors.Wrap(err, "calculating tx merkle root")
	}

	return b, nil
}
