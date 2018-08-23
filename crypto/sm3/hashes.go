package sm3

import (
	"io"
	"sync"
)

var pool = &sync.Pool{New: func() interface{} { return New() }}

// ShakeHash defines the interface to hash functions that
// support arbitrary-length output.
type ShakeHash interface {
	// Write absorbs more data into the hash's state. It panics if input is
	// written to it after output has been read from it.
	io.Writer

	// Read reads more output from the hash; reading affects the hash's
	// state. (ShakeHash.Read is thus very different from Hash.Sum)
	// It never returns an error.
	io.Reader

	// Clone returns a copy of the ShakeHash in its current state.
	Clone() ShakeHash

	// Reset resets the ShakeHash to its initial state.
	Reset()
}

// Get256 returns an initialized SHA3-256 hash ready to use.
// It is like sha3.New256 except it uses the freelist.
// The caller should call Put256 when finished with the returned object.
func Get256() ShakeHash {
	return pool.Get().(ShakeHash)
}

// Put256 resets h and puts it in the freelist.
func Put256(h ShakeHash) {
	h.Reset()
	pool.Put(h)
}

// Sum256 returns the SM3 digest of the data.
func Sum256(data []byte) (digest [32]byte) {
	hash := Sm3Sum(data)
	copy(digest[:], hash)
	return
}

func Sum(hash, data []byte) {
	tmp := Sm3Sum(data)
	copy(hash, tmp[:])
}
