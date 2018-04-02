package test

import (
	"testing"
)

func TestChain(t *testing.T) {
	walk(t, chainTestDir, func(t *testing.T, name string, test *ChainTestConfig) {
		if err := test.Run(); err != nil {
			t.Fatal(err)
		}
	})
}