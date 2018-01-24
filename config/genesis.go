package config

import (
	"fmt"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/consensus/aihash"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

// GenerateGenesisTx will return genesis transaction
func GenerateGenesisTx() *legacy.Tx {
	txData := legacy.TxData{
		Version:        1,
		SerializedSize: 63,
		Inputs:         []*legacy.TxInput{},
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

	var hash128 [128]*bc.Hash
	for i := 0; i < 128 ; i++ {
		hash := bc.NewHash(seed)
		hash128[i] = &hash
	}
	aihash.Notify(hash128)
	block := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:     1,
			Height:      0,
			Nonce:       0,
			Seed:        bc.BytesToHash(aihash.Md.GetSeed()),
			TimestampMS: 1511318565142,
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
			},
			Bits: 99900000,
		},
		Transactions: []*legacy.Tx{genesisCoinbaseTx},
	}

	fmt.Printf("1----------block:%v\n", block)
	for {
		hash := block.Hash()
		proofHash, err := aihash.AIHash(&hash, aihash.Md.GetCache())
		if err != nil {
			log.Panicf("Fatal AIHash")
		}

		if difficulty.CheckProofOfWork(proofHash, block.Bits) {
			fmt.Printf("Nonce----------nonce:%v\n", block.Nonce)
			log.Info(block.Nonce)
			break
		}
		block.Nonce++
	}
	fmt.Printf("2----------block:%v\n", block)
	return block
}
