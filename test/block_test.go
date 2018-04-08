package test

import (
	"testing"
)

func TestBlockchain(t *testing.T) {
	walk(t, blockchainTestDir, func(t *testing.T, name string, test *BlockchainTestConfig) {
		if err := test.Run(); err != nil {
			t.Fatal(err)
		}
	})
}
