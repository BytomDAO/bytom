package test

import (
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
)

func NewBlock(chain *protocol.Chain, txs []*types.Tx, controlProgram []byte) (*types.Block, error) {
	view := state.NewUtxoViewpoint()
	gasUsed := uint64(0)
	txsFee := uint64(0)
	txEntries := []*bc.Tx{nil}
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)

	preBlock := chain.BestBlock()
	preBcBlock := types.MapBlock(preBlock)

	var compareDiffBH *types.BlockHeader
	if compareDiffBlock, err := chain.GetBlockByHeight(preBlock.Height - consensus.BlocksPerRetarget); err == nil {
		compareDiffBH = &compareDiffBlock.BlockHeader
	}

	b := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            preBlock.Height + 1,
			PreviousBlockHash: preBlock.Hash(),
			Timestamp:         uint64(time.Now().Unix()),
			BlockCommitment:   types.BlockCommitment{},
			Bits:              difficulty.CalcNextRequiredDifficulty(&preBlock.BlockHeader, compareDiffBH),
		},
		Transactions: []*types.Tx{nil},
	}
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: preBlock.Height + 1}}

	for _, tx := range txs {
		gasOnlyTx := false
		if err := chain.GetTransactionsUtxo(view, []*bc.Tx{tx.Tx}); err != nil {
			continue
		}

		gasStatus, err := validation.ValidateTx(tx.Tx, preBcBlock)
		if err != nil {
			if !gasStatus.GasVaild {
				continue
			}
			gasOnlyTx = true
		}

		if gasUsed+uint64(gasStatus.GasUsed) > consensus.MaxBlockGas {
			break
		}

		if err := view.ApplyTransaction(bcBlock, tx.Tx, gasOnlyTx); err != nil {
			continue
		}

		txStatus.SetStatus(len(b.Transactions), gasOnlyTx)
		b.Transactions = append(b.Transactions, tx)
		txEntries = append(txEntries, tx.Tx)
		gasUsed += uint64(gasStatus.GasUsed)
		if gasUsed == consensus.MaxBlockGas {
			break
		}
		txsFee += txFee(tx)
	}

	coinbaseTx, err := CreateCoinbaseTx(controlProgram, preBlock.Height+1, txsFee)
	if err != nil {
		return nil, err
	}

	b.Transactions[0] = coinbaseTx
	txEntries[0] = coinbaseTx.Tx
	b.TransactionsMerkleRoot, err = bc.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.TransactionStatusHash, err = bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	return b, err
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

func SolveAndUpdate(chain *protocol.Chain, block *types.Block) error {
	seed, err := chain.GetSeed(block.Height, &block.PreviousBlockHash)
	if err != nil {
		return err
	}
	Solve(seed, block)
	if err := chain.SaveBlock(block); err != nil {
		return err
	}
	if err := chain.ConnectBlock(block); err != nil {
		return err
	}
	return nil
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
