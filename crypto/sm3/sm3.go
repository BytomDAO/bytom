package sm3

import "github.com/bytom/bytom/crypto"

func Sum256(data []byte) (digest [32]byte) {
	hash := crypto.Sm3(data)
	copy(digest[:], hash)
	return
}
