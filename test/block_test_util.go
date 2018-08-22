package test

import (
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm"
)

// NewBlock create block according to the current status of chain
func NewBlock(chain *protocol.Chain, txs []*types.Tx, controlProgram []byte) (*types.Block, error) {
	gasUsed := uint64(0)
	txsFee := uint64(0)
	txEntries := []*bc.Tx{nil}
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)

	preBlockHeader := chain.BestBlockHeader()
	preBlockHash := preBlockHeader.Hash()
	nextBits, err := chain.CalcNextBits(&preBlockHash)
	if err != nil {
		return nil, err
	}

	b := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            preBlockHeader.Height + 1,
			PreviousBlockHash: preBlockHeader.Hash(),
			Timestamp:         preBlockHeader.Timestamp + 1,
			BlockCommitment:   types.BlockCommitment{},
			Bits:              nextBits,
		},
		Transactions: []*types.Tx{nil},
	}

	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: preBlockHeader.Height + 1}}
	for _, tx := range txs {
		gasOnlyTx := false
		gasStatus, err := validation.ValidateTx(tx.Tx, bcBlock)
		if err != nil {
			if !gasStatus.GasValid {
				continue
			}
			gasOnlyTx = true
		}

		txStatus.SetStatus(len(b.Transactions), gasOnlyTx)
		b.Transactions = append(b.Transactions, tx)
		txEntries = append(txEntries, tx.Tx)
		gasUsed += uint64(gasStatus.GasUsed)
		txsFee += txFee(tx)
	}

	coinbaseTx, err := CreateCoinbaseTx(controlProgram, preBlockHeader.Height+1, txsFee)
	if err != nil {
		return nil, err
	}

	b.Transactions[0] = coinbaseTx
	txEntries[0] = coinbaseTx.Tx
	b.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.TransactionStatusHash, err = types.TxStatusMerkleRoot(txStatus.VerifyStatus)
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
		if err := SolveAndUpdate(chain, block); err != nil {
			return err
		}
	}
	return nil
}

// SolveAndUpdate solve difficulty and update chain status
func SolveAndUpdate(chain *protocol.Chain, block *types.Block) error {
	seed, err := chain.CalcNextSeed(&block.PreviousBlockHash)
	if err != nil {
		return err
	}
	Solve(seed, block)
	_, err = chain.ProcessBlock(block)
	return err
}

// Solve simulate solve difficulty by add result to cache
func Solve(seed *bc.Hash, block *types.Block) {
	hash := block.BlockHeader.Hash()
	tensority.AIHash.AddCache(&hash, seed, &bc.Hash{})
}
