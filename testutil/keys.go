package testutil

import (
	sm2util "github.com/bytom/bytom/crypto/sm2"
	"github.com/bytom/bytom/crypto/sm2/chainkd"
)

var (
	TestXPub chainkd.XPub
	TestXPrv chainkd.XPrv
	TestPub  sm2util.PubKey
	TestPubs []sm2util.PubKey
)

type zeroReader struct{}

func (z zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func init() {
	var err error
	TestXPrv, TestXPub, err = chainkd.NewXKeys(zeroReader{})
	if err != nil {
		panic(err)
	}
	TestPub = TestXPub.PublicKey()
	TestPubs = []sm2util.PubKey{TestPub}
}
