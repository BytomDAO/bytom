// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm/vmutil"
)

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func createCoinbaseTx(accountManager *account.Manager, amount uint64, blockHeight uint64) (tx *types.Tx, err error) {
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
	if err = builder.AddInput(types.NewCoinbaseInput(
		append([]byte{0x00}, []byte(strconv.FormatUint(blockHeight, 10))...),
	), &txbuilder.SigningInstruction{}); err != nil {
		return
	}
	if err = builder.AddOutput(types.NewTxOutput(*consensus.BTMAssetID, amount, script)); err != nil {
		return
	}
	_, txData, err := builder.Build()
	if err != nil {
		return
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return
	}
	txData.SerializedSize = uint64(len(byteData))

	tx = &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}
	return
}

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(c *protocol.Chain, txPool *protocol.TxPool, accountManager *account.Manager) (b *types.Block, err error) {
	view := state.NewUtxoViewpoint()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txEntries := []*bc.Tx{nil}
	gasUsed := uint64(0)
	txFee := uint64(0)

	// get preblock info for generate next block
	preBlockHeader := c.BestBlockHeader()
	preBlockHash := preBlockHeader.Hash()
	nextBlockHeight := preBlockHeader.Height + 1
	nextBits, err := c.CalcNextBits(&preBlockHash)
	if err != nil {
		return nil, err
	}

	b = &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            nextBlockHeight,
			PreviousBlockHash: preBlockHash,
			Timestamp:         uint64(time.Now().Unix()),
			BlockCommitment:   types.BlockCommitment{},
			Bits:              nextBits,
		},
	}
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: nextBlockHeight}}
	b.Transactions = []*types.Tx{nil}

	txs := txPool.GetTransactions()
	sort.Sort(byTime(txs))
	for _, txDesc := range txs {
		tx := txDesc.Tx.Tx
		gasOnlyTx := false

		if err := c.GetTransactionsUtxo(view, []*bc.Tx{tx}); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			continue
		}

		gasStatus, err := validation.ValidateTx(tx, bcBlock)
		if err != nil {
			if !gasStatus.GasValid {
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

	b.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = bc.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.BlockHeader.BlockCommitment.TransactionStatusHash, err = bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	return b, err
}
