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
	"github.com/bytom/crypto/scrypt"
	"github.com/bytom/protocol/bc"
)

const (
	matSize     = 1 << 8 // Size of matrix
	matNum      = 1 << 8 // Number of matrix
	epochLength = 1 << 7 // Blocks per epoch
	mulRounds   = 1 << 8 // Number of rounds in mulmatrix
)

type miningData struct {
	seed  []byte   // seed length is 32 Bytes
	cache []uint32 // cache length is 256*256*256 = 16777216 Bytes
}

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

// generateSeed is the seed to use for generating a verification cache.
func (md *miningData) generateSeed(blockHashs []*bc.Hash) {
	seed := make([]byte, 32)
	seedSlice := make([]byte, 0)

	for _, blockHash := range blockHashs {
		seedSlice = append(seedSlice, blockHash.Bytes()...)
	}
	sha256 := makeHasher(sha3.New256())
	sha256(seed, seedSlice)

	md.seed = seed
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
func (md *miningData) generateCache() {
	extSeed := extendSeed(md.seed)
	cache := make([]uint32, 0)

	xy := make([]uint32, 64)
	v := make([]uint32, 32*1024)
	for i := 0; i < 128; i++ {
		scrypt.Smix(extSeed, 1, 1024, v, xy)
		cache = append(cache, v...)
	}

	md.cache = cache
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

// fnv is an algorithm inspired by the FNV hash, which in some cases is used as
// a non-associative substitute for XOR. Note that we multiply the prime with
// the full 32-bit input, in contrast with the FNV-1 spec which multiplies the
// prime with one byte (octet) in turn.
func fnv(a, b uint32) uint32 {
	return a*0x01000193 ^ b
}

func mulMatrix(cache []uint32, headerhash []byte) []uint8 {
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

	result := make([]uint8, 0)
	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize; j++ {
			result = append(result, uint8(mc.At(i, j)))
		}
	}

	return result
}

func hashMatrix(result []uint8) *bc.Hash {
	var mat8 [matSize][matSize]uint8
	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize; j++ {
			mat8[i][j] = result[i*matSize+j]
		}
	}

	var mat32 [matSize][matSize / 4]uint32
	// ATTENTION !!!!!!! C++ is different!!!
	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize; j += 4 {
			mat32[i][j] = (uint32(mat8[i][j])) |
				((uint32(mat8[i][j+1])) << 8) |
				((uint32(mat8[i][j+2])) << 16) |
				((uint32(mat8[i][j+3])) << 24)
		}
	}

	data := make([]uint32, 0)
	for k := matSize; k > 1; k = k / 2 {
		for j := 0; j < k/2; j++ {
			for i := 0; i < matSize/4; i++ {
				mat32[j][i] = fnv(mat32[j][i], mat32[j+k/2][i])
			}
		}
	}

	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize/4; j++ {
			data = append(data, mat32[i][j])
		}
	}

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&data))
	header.Len *= 4
	header.Cap *= 4
	dataBytes := *(*[]byte)(unsafe.Pointer(&header))

	bch := bc.NewHash(sha3.Sum256(dataBytes))

	return &bch
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
