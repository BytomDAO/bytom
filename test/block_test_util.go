package test

import (
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

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
