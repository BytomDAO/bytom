// +build gm

package sm3

import (
	"github.com/tjfoc/gmsm/sm3"
)

// Sum256 returns the SM3 digest of the data.
func Sum256(data []byte) (digest [32]byte) {
	hash := sm3.Sm3Sum(data)
	copy(digest[:], hash)
	return
}

// Sum calculate data into hash
func Sum(hash, data []byte) {
	tmp := sm3.Sm3Sum(data)
	copy(hash, tmp)
}
