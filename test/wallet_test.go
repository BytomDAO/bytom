// +build functional

package test

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestWallet(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	walk(t, walletTestDir, func(t *testing.T, name string, test *walletTestConfig) {
		if err := test.Run(); err != nil {
			t.Fatal(err)
		}
	})
}
