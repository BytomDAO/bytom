package config

import (
	"fmt"
	"log"
	"testing"

	"github.com/bytom/consensus/aihash"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
)

// test genesis
func TestGenerateGenesisTx(t *testing.T) {
	if tx := GenerateGenesisTx(); tx == nil {
		t.Errorf("Generate genesis tx failed")
	}
}

func TestGenerateGenesisBlock(t *testing.T) {
	block := GenerateGenesisBlock()

	var hash128 [128]*bc.Hash
	for i := 0; i < 128; i++ {
		hash := bc.NewHash(block.Seed.Byte32())
		hash128[i] = &hash
	}
	aihash.Notify(hash128)

	for {
		hash := block.Hash()
		proofHash, err := aihash.AIHash(&hash, aihash.Md.GetCache())
		if err != nil {
			log.Panicf("Fatal AIHash")
		}

		if difficulty.CheckProofOfWork(proofHash, block.Bits) {
			break
		}
		block.Nonce++
	}
	fmt.Print("block nonce: ", block.Nonce)
}
