package initblocks

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func initBlocks(asset2distributions map[string][]AddressBalance) []*types.Block {
	var blocks []*types.Block
	var height uint64
	var preBlockHash bc.Hash

	allTxs := buildAllTxs(asset2distributions)
	for i := 0; i < len(allTxs); i += TxCntPerBlock {
		batchTxs := getBatchTxs(allTxs, i, TxCntPerBlock)
		block := buildBlock(batchTxs, preBlockHash, height)
		blocks = append(blocks, block)

		preBlockHash = block.Hash()
		height++
	}

	return blocks
}

func getBatchTxs(allTxs []*types.Tx, offset, limit int) []*types.Tx {
	if offset > len(allTxs) {
		return nil
	}

	if len(allTxs[offset:]) < limit {
		return allTxs[offset:]
	}

	return allTxs[offset : offset+limit]
}

func buildBlock(txs []*types.Tx, preBlockHash bc.Hash, height uint64) *types.Block {
	bcTxs := make([]*bc.Tx, 0, len(txs))
	for _, tx := range txs {
		bcTxs = append(bcTxs, tx.Tx)
	}

	merkleRoot, err := types.TxMerkleRoot(bcTxs)
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            height,
			Timestamp:         1524549600000,
			PreviousBlockHash: preBlockHash,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
			},
		},
		Transactions: txs,
	}
	return block
}
