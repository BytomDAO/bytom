package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// GenerateGenesisTx will return genesis transaction
func GenerateGenesisTx() *types.Tx {
	txData := types.TxData{
		Version:        1,
		SerializedSize: 63,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("May 4th Be With You")),
		},
		Outputs: []*types.TxOutput{
			&types.TxOutput{
				AssetVersion: 1,
				OutputCommitment: types.OutputCommitment{
					AssetAmount: bc.AssetAmount{
						AssetId: consensus.BTMAssetID,
						Amount:  consensus.InitialBlockSubsidy,
					},
					VMVersion:      1,
					ControlProgram: []byte{81},
				},
			},
		},
	}

	return types.NewTx(txData)
}

// GenerateGenesisBlock will return genesis block
func GenerateGenesisBlock() *types.Block {
	genesisCoinbaseTx := GenerateGenesisTx()
	merkleRoot, err := bc.TxMerkleRoot([]*bc.Tx{genesisCoinbaseTx.Tx})
	if err != nil {
		log.Panicf("Fatal create tx merkelRoot")
	}

	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, err := bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("Fatal create tx status gmerkelRoot")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     4216221,
			Timestamp: 1516788453,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
			Bits: 2305843009214532812,
		},
		Transactions: []*types.Tx{genesisCoinbaseTx},
	}
	return block
}
