package config

import (
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
)

// test genesis
func TestGenerateGenesisTx(t *testing.T) {
	if tx := GenerateGenesisTx(); tx == nil {
		t.Errorf("Generate genesis tx failed")
	}
}

func TestGenerateGenesisBlock(t *testing.T) {
	block := GenerateGenesisBlock()
	nonce := block.Nonce
	for {
		hash := block.Hash()
		if difficulty.CheckProofOfWork(&hash, consensus.InitialSeed, block.Bits) {
			break
		}
		block.Nonce++
	}
	if block.Nonce != nonce {
		t.Errorf("correct nonce is %d, but get %d", block.Nonce, nonce)
	}
}
