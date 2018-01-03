package config

import (
	"testing"
)

// test genesis
func TestGenerateGenesisTx(t *testing.T) {
	if tx := GenerateGenesisTx(); tx == nil {
		t.Errorf("Generate genesis tx failed")
	}
}

func TestGenerateGenesisBlock(t *testing.T) {
	if block := GenerateGenesisBlock(); block == nil {
		t.Errorf("Generate genesis block failed")
	}
}
