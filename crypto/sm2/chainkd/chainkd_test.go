package chainkd

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/bytom/crypto/sm2"
)

func TestSign(t *testing.T) {
	fmt.Println("====TestSign====")
	d, _ := hex.DecodeString("b54684dd7f7a66ee2e9a0a7cbd7f2efdc3c22b117812795e08e3d19e5744483ab54684dd7f7a66ee2e9a0a7cbd7f2efdc3c22b117812795e08e3d19e5744483a")
	var xpriv XPrv
	copy(xpriv[:], d[:])
	sig := xpriv.Sign(d[:32])
	fmt.Printf("sig: %x\n", sig[:])

	xpub := xpriv.XPub()
	r := sm2.VerifyCompressedPubkey(xpub[:33], d[:32], sig[:])
	fmt.Println("verify sigature!")
	fmt.Printf("publickey: %x\n", xpub[:33])
	fmt.Printf("msg: %x\n", d[:32])
	fmt.Printf("signature: %x\n", sig[:32])
	fmt.Printf("result: %v\n", r)
}
