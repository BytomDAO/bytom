package test

import (
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/validation"
	"github.com/bytom/bytom/protocol/vm"
)

// NewBlock create block according to the current status of chain
func NewBlock(chain *protocol.Chain, txs []*types.Tx, controlProgram []byte) (*types.Block, error) {
	gasUsed := uint64(0)
	txsFee := uint64(0)

	preBlockHeader := chain.BestBlockHeader()

	b := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            preBlockHeader.Height + 1,
			PreviousBlockHash: preBlockHeader.Hash(),
			Timestamp:         preBlockHeader.Timestamp + 10000,
			BlockCommitment:   types.BlockCommitment{},
		},
		Transactions: []*types.Tx{nil},
	}

	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: preBlockHeader.Height + 1}}
	for _, tx := range txs {
		gasStatus, err := validation.ValidateTx(tx.Tx, bcBlock, chain.ProgramConverter)
		if err != nil {
			continue
		}

		b.Transactions = append(b.Transactions, tx)
		gasUsed += uint64(gasStatus.GasUsed)
		txsFee += txFee(tx)
	}

	coinbaseTx, err := CreateCoinbaseTx(controlProgram, preBlockHeader.Height+1, txsFee)
	if err != nil {
		return nil, err
	}

	b.Transactions[0] = coinbaseTx
	if len(txs) > 0 {
		b.Transactions = append(b.Transactions, txs...)
	}

	txEntries := []*bc.Tx{nil}
	txEntries[0] = coinbaseTx.Tx
	for _, tx := range txs {
		txEntries = append(txEntries, tx.Tx)
	}
	b.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	return b, err
}

// ReplaceCoinbase replace the coinbase tx of block with coinbaseTx
func ReplaceCoinbase(block *types.Block, coinbaseTx *types.Tx) (err error) {
	block.Transactions[0] = coinbaseTx
	txEntires := []*bc.Tx{coinbaseTx.Tx}
	for i := 1; i < len(block.Transactions); i++ {
		txEntires = append(txEntires, block.Transactions[i].Tx)
	}

	block.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntires)
	return
}

// AppendBlocks append empty blocks to chain, mainly used to mature the coinbase tx
func AppendBlocks(chain *protocol.Chain, num uint64) error {
	for i := uint64(0); i < num; i++ {
		block, err := NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
		if err != nil {
			return err
		}
		if _, err := chain.ProcessBlock(block); err != nil {
			return err
		}
	}
	return nil
}
