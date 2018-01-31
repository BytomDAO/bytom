package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

// GenerateGenesisTx will return genesis transaction
func GenerateGenesisTx() *legacy.Tx {
	txData := legacy.TxData{
		Version:        1,
		SerializedSize: 63,
		Inputs: []*legacy.TxInput{
			legacy.NewCoinbaseInput([]byte("May 4th Be With You"), nil),
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

	var seed [32]byte
	sha3pool.Sum256(seed[:], make([]byte, 32))

	block := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     4216077,
			Seed:      bc.NewHash(seed),
			Timestamp: 1516788453,
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
			},
			Bits: 2161727821138738707,
			TransactionStatus: bc.TransactionStatus{
				Bitmap: []byte{0},
			},
		},
		Transactions: []*legacy.Tx{genesisCoinbaseTx},
	}

	for {
		hash := block.Hash()
		if difficulty.CheckProofOfWork(&hash, block.Bits) {
			log.WithField("block.Nonce=", block.Nonce).Info()
			break
		}
		block.Nonce++
	}
	return block
}
