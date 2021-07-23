package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func toBCTxs(txs []*types.Tx) []*bc.Tx {
	var bcTxs []*bc.Tx
	for _, tx := range txs {
		bcTxs = append(bcTxs, tx.Tx)
	}
	return bcTxs
}

func mainNetGenesisBlock() *types.Block {
	txs := GenesisTxs()
	merkleRoot, err := types.TxMerkleRoot(toBCTxs(txs))
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1524549600000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
			},
		},
		Transactions: txs,
	}
	return block
}

func testNetGenesisBlock() *types.Block {
	txs := GenesisTxs()
	merkleRoot, err := types.TxMerkleRoot(toBCTxs(txs))
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
			},
		},
		Transactions: txs,
	}
	return block
}

func soloNetGenesisBlock() *types.Block {
	txs := GenesisTxs()
	merkleRoot, err := types.TxMerkleRoot(toBCTxs(txs))
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
			},
		},
		Transactions: txs,
	}
	return block
}

// GenesisBlock will return genesis block
func GenesisBlock() *types.Block {
	return map[string]func() *types.Block{
		"main": mainNetGenesisBlock,
		"test": testNetGenesisBlock,
		"solo": soloNetGenesisBlock,
	}[consensus.ActiveNetParams.Name]()
}
