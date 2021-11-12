package sm2

import (
	"math/big"

	"github.com/tjfoc/gmsm/sm2"
)

// VerifyCompressedPubkey verify sigature is valid.
// The parameters is bytes.
// The publickey is compressed, the length is 33 bytes.
func VerifyCompressedPubkey(compressedPublicKey, hash, signature []byte) bool {
	pubkey := sm2.Decompress(compressedPublicKey)
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	return sm2.Verify(pubkey, hash, r, s)
}
