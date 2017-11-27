package config

import (
	"testing"
)

// test genesis
func TestGenesis(t *testing.T) {
	if tx := GenerateGenesisTx(); tx == nil {
		t.Errorf("Generate genesis tx failed")
	}

	if block := GenerateGenesisBlock(); block == nil {
		t.Errorf("Generate genesis block failed")
	}
}
