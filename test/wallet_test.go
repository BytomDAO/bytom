// +build functional

package test

import (
	"testing"
)

func TestWallet(t *testing.T) {
	walk(t, walletTestDir, func(t *testing.T, name string, test *walletTestConfig) {
		if err := test.Run(); err != nil {
			t.Fatal(err)
		}
	})
}
