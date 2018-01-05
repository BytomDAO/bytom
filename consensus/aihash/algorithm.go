package aihash

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"reflect"
	"sync/atomic"
	"unsafe"

	"golang.org/x/crypto/sha3"
	"gonum.org/v1/gonum/mat"

	"github.com/bytom/common/bitutil"
	"github.com/bytom/consensus/aihash/matrix"
	"github.com/bytom/crypto/scrypt"
	"github.com/bytom/protocol/bc"
)

const (
	hashBytes   = 64 // Hash length in bytes
	cacheRounds = 3  // Number of rounds in cache production
)

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
type hasher func(dest []byte, data []byte)

// makeHasher creates a repetitive hasher, allowing the same hash data structures
// to be reused between hash runs instead of requiring new ones to be created.
// The returned function is not thread safe!
func makeHasher(h hash.Hash) hasher {
	return func(dest []byte, data []byte) {
		h.Write(data)
		h.Sum(dest[:0])
		h.Reset()
	}
}

// createSeed is the seed to use for generating a verification cache.
func createSeed(preSeed *bc.Hash, blockHashs []*bc.Hash) []byte {
	seed := make([]byte, 32)
	seedSlice := preSeed.Bytes()

	for _, blockHash := range blockHashs {
		seedSlice = append(seedSlice, blockHash.Bytes()...)
	}
	sha256 := makeHasher(sha3.New256())
	sha256(seed, seedSlice)

	return seed
}

// extend seed from 32 byte to 128 byte
func extendSeed(seed []byte) []byte {
	extSeed := make([]byte, 128)
	extSeed[:32] = seed

	for i := 0; i < 3; i++ {
		h := sha3.Sum256(extSeed[i*32 : (i+1)*32])
		copy(extSeed[(i+1)*32:(i+2)*32], h[:])
	}

	return extSeed
}

// seed length is 32 bytes, cache is 16MB.
func generateCache(seed []byte) []uint32 {
	extSeed := extendSeed(seed)
	cache := make([]uint32, 0)

	xy := make([]uint32, 64)
	v := make([]uint32, 32*1024)
	for i := 0; i < 128; i++ {
		scrypt.Smix(extSeed, 1, 1024, v, xy)
		cache = append(cache, v...)
	}

	return cache
}

// extendHash extend 32 Bytes hash to 256 Bytes.
func extendHash(hash []byte) []byte {
	extHash := make([]byte, 256)
	copy(extHash[:32], hash)

	for i := 1; i < 8; i++ {
		h := sha256.Sum256(extHash[(i-1)*32 : i*32])
		copy(extHash[i*32:(i+1)*32], h[:])
	}

	return extHash
}

func mulMatrix(cache []uint32, headerhash []byte) []byte {
	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&cache))
	header.Len *= 4
	header.Cap *= 4
	cacheInt8 := *(*[]int8)(unsafe.Pointer(&header))

	data := make([]float64, matNum*matSize*matSize)
	for i := 0; i < len(cacheInt8); i++ {
		data[i] = float64(cacheInt8[i])
	}

	exthash := extendHash(headerhash)

	mb := mat.NewDense(matSize, matSize, data[:matSize*matSize])
	mc := mat.NewDense(matSize, matSize, make([]float64, matSize*matSize))
	for i := 0; i < 256; i++ {
		index := int(exthash[i])
		ma := mat.NewDense(matSize, matSize, data[index*matSize*matSize:(index+1)*matSize*matSize])
		mc.Mul(ma, mb)

		for j := 0; j < matSize; j++ {
			for k := 0; k < matSize; k++ {
				i32v := int32(mc.At(j, k))
				i8v := int8((i32v & 0xff) +
					((i32v >> 8) & 0xff))
				mc.Set(j, k, float64(i8v))
			}
		}

		mb = mc
	}

	result := make([]byte, 0)
	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize; j++ {
			result = append(result, byte(mc.At(i, j)))
		}
	}

	return result
}

// isLittleEndian returns whether the local system is running in little or big
// endian byte order.
func isLittleEndian() bool {
	n := uint32(0x01020304)
	return *(*byte)(unsafe.Pointer(&n)) == 0x04
}

// swap changes the byte order of the buffer assuming a uint32 representation.
func swap(buffer []byte) {
	for i := 0; i < len(buffer); i += 4 {
		binary.BigEndian.PutUint32(buffer[i:], binary.LittleEndian.Uint32(buffer[i:]))
	}
}

// prepare converts an btmhash cache or dataset from a byte stream into the internal
// int representation. All btmhash methods work with ints to avoid constant byte to
// int conversions as well as to handle both little and big endian systems.
func prepare(dest []uint32, src []byte) {
	for i := 0; i < len(dest); i++ {
		dest[i] = binary.LittleEndian.Uint32(src[i*4:])
	}
}

func bytesToUint32(src []byte) []uint32 {
	dest := make([]uint32, len(src)/4)
	for i := 0; i < len(dest); i++ {
		dest[i] = binary.LittleEndian.Uint32(src[i*4:])
	}

	return dest
}

func hashMatrix(m *matrix.Matrix, matSize int) *bc.Hash {
	var item []byte
	for i := 1; i <= matSize; i++ {
		for j := 1; j <= matSize; j++ {
			item = append(item, byte(m.Get(i, j)))
		}
	}

	bch := bc.NewHash(sha3.Sum256(item))

	return &bch
}

var primes = []int{
	3, 5, 7, 11, 13, 17, 19, 23,
	29, 31, 37, 41, 43, 47, 53, 59,
	61, 67, 71, 73, 79, 83, 89, 97,
	101, 103, 107, 109, 113, 127, 131, 137,
	139, 149, 151, 157, 163, 167, 173, 179,
	181, 191, 193, 197, 199, 211, 223, 227,
	229, 233, 239, 241, 251, 257, 263, 269,
	271, 277, 281, 283, 293, 307, 311, 313,
	317, 331, 337, 347, 349, 353, 359, 367,
	373, 379, 383, 389, 397, 401, 409, 419,
	421, 431, 433, 439, 443, 449, 457, 461,
	463, 467, 479, 487, 491, 499, 503, 509,
	521, 523, 541, 547, 557, 563, 569, 571,
	577, 587, 593, 599, 601, 607, 613, 617,
	619, 631, 641, 643, 647, 653, 659, 661,
	673, 677, 683, 691, 701, 709, 719, 727}
