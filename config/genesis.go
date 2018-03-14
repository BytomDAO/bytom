package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

// GenerateGenesisTx will return genesis transaction
func GenerateGenesisTx() *legacy.Tx {
	txData := legacy.TxData{
		Version:        1,
		SerializedSize: 63,
		Inputs: []*legacy.TxInput{
			legacy.NewCoinbaseInput([]byte("May 4th Be With You")),
		},
		Outputs: []*legacy.TxOutput{
			&legacy.TxOutput{
				AssetVersion: 1,
				OutputCommitment: legacy.OutputCommitment{
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

	return legacy.NewTx(txData)
}

// GenerateGenesisBlock will return genesis block
func GenerateGenesisBlock() *legacy.Block {
	genesisCoinbaseTx := GenerateGenesisTx()
	merkleRoot, err := bc.MerkleRoot([]*bc.Tx{genesisCoinbaseTx.Tx})
	if err != nil {
		log.Panicf("Fatal create merkelRoot")
	}
	txStatus := bc.NewTransactionStatus()

	block := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     4216083,
			Timestamp: 1516788453,
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  bc.EntryID(txStatus),
			},
			Bits: 2305843009222082559,
		},
		Transactions: []*legacy.Tx{genesisCoinbaseTx},
	}
	return block
}
