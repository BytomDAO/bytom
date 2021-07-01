package config

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func GenesisTx() *types.Tx {
	contract, err := hex.DecodeString("00148c9d063ff74ee6d9ffa88d83aeb038068366c4c4")
	if err != nil {
		log.Panicf("fail on decode genesis tx output control program")
	}

	txData := types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(*consensus.BTMAssetID, 0, contract, nil),
		},
	}
	return types.NewTx(txData)
}

func mainNetGenesisBlock() *types.Block {
	tx := GenesisTx()
	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
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
		Transactions: []*types.Tx{tx},
	}
	return block
}

func testNetGenesisBlock() *types.Block {
	tx := GenesisTx()
	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
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
		Transactions: []*types.Tx{tx},
	}
	return block
}

func soloNetGenesisBlock() *types.Block {
	tx := GenesisTx()
	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
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
		Transactions: []*types.Tx{tx},
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
