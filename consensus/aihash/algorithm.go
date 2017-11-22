package aihash

import (
	"encoding/binary"
	"fmt"
	"hash"
	"reflect"
	"sync/atomic"
	"unsafe"

	//log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/common/bitutil"
	"github.com/bytom/consensus/aihash/matrix"
	// "github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

const (
	epochLength = 128 // Blocks per epoch
	hashBytes   = 64  // Hash length in bytes
	cacheRounds = 3   // Number of rounds in cache production
)

const (
	matSize     = 1 << 9 // Size of matrix
	matNum      = 1 << 7 // Number of matrix
	round       = 10
	cacheLength = matSize * matSize * matNum // Bytes of cache production
)

var (
	nonce []uint64 = make([]uint64, 4)
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
func createSeed(preSeed *bc.Hash, preEpochBlockHash []*bc.Hash) []byte {
	seed := make([]byte, 32)
	var ss []byte
	ss = append(ss, (*preSeed).Bytes()...)

	for _, h := range preEpochBlockHash {
		ss = append(ss, (*h).Bytes()...)
	}
	sha256 := makeHasher(sha3.New256())
	sha256(seed, ss)

	return seed
}

// convert []byte to *bc.Hash, the lenght is 32 bytes.
func bytesToPointerHash(src []byte) *bc.Hash {
	var arr [32]byte
	for i, v := range src {
		arr[i] = v
	}
	newhash := bc.NewHash(arr)

	return &newhash
}

// convert []byte to []*bc.Hash
func bytesToSlicePointerHash(src []byte) []*bc.Hash {
	var sbh []*bc.Hash
	for i := 0; i < len(src)/32; i++ {
		s := src[i*32 : (i+1)*32]
		var arr [32]byte

		for j, v := range s {
			arr[j] = v
		}
		newhash := bc.NewHash(arr)
		sbh = append(sbh, &newhash)
	}

	return sbh
}

// This method places the result into dest in machine byte order.
func generateCache(dest []uint32, seed []byte) {
	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&dest))
	header.Len *= 4
	header.Cap *= 4
	cache := *(*[]byte)(unsafe.Pointer(&header))

	// Calculate the number of thoretical rows (we'll store in one buffer nonetheless)
	size := uint64(len(cache))
	rows := int(size) / hashBytes

	// Start a monitoring goroutine to report progress on low end devices
	var progress uint32

	done := make(chan struct{})
	defer close(done)

	sha512 := makeHasher(sha3.New512())

	// Sequentially produce the initial dataset
	sha512(cache, seed)
	for offset := uint64(hashBytes); offset < size; offset += hashBytes {
		sha512(cache[offset:], cache[offset-hashBytes:offset])
		atomic.AddUint32(&progress, 1)
	}
	// Use a low-round version of randmemohash
	temp := make([]byte, hashBytes)

	for i := 0; i < cacheRounds; i++ {
		for j := 0; j < rows; j++ {
			var (
				srcOff = ((j - 1 + rows) % rows) * hashBytes
				dstOff = j * hashBytes
				xorOff = (binary.LittleEndian.Uint32(cache[dstOff:]) % uint32(rows)) * hashBytes
			)
			bitutil.XORBytes(temp, cache[srcOff:srcOff+hashBytes], cache[xorOff:xorOff+hashBytes])
			sha512(cache[dstOff:], temp)

			atomic.AddUint32(&progress, 1)
		}
	}
	// Swap the byte order on big endian systems and return
	if !isLittleEndian() {
		swap(cache)
	}
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

// fill the matrix list
func fillMatrixList(matList []matrix.Matrix, cache []uint32, height uint64) {
	var locationIndex uint64 = (height - 1) % epochLength

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&cache))
	header.Len *= 4
	header.Cap *= 4
	cacheInt8 := *(*[]int8)(unsafe.Pointer(&header))

	// fmt.Println("cacheInt8 is:", cacheInt8)

	for i := 0; i < matNum; i++ {
		startIndex := (matSize*matSize*i + location[locationIndex]) % cacheLength
		endIndex := (matSize*matSize*(i+1) + location[locationIndex]) % cacheLength
		if startIndex < endIndex {
			matList[i] = matrix.New(matSize, matSize, cacheInt8[startIndex:endIndex])
		} else {
			matrixData := make([]int8, matSize*matSize)
			copy(matrixData, cacheInt8[startIndex:])
			copy(matrixData[cacheLength-startIndex:], cacheInt8[:endIndex+1])
			matList[i] = matrix.New(matSize, matSize, matrixData)
		}
	}

	// fmt.Println("Print the matrix!")

	// for i := 0; i < matNum; i++ {
	// 	matList[i].Print()
	// }

	// fmt.Println("locationIndex is:", locationIndex)
	// fmt.Println("location[block % epochLength] is:", location[(height-1)%epochLength])
}

func mulMatrix(matList []matrix.Matrix, matIndex []byte) *matrix.Matrix {
	var index uint8
	ma := matrix.Zeros(matSize, matSize)
	mb := matrix.Zeros(matSize, matSize)

	for i := 1; i <= matSize; i++ {
		for j := 1; j <= matSize; j++ {
			mb.Set(i, j, matList[0].Get(i, j))
		}
	}

	indexnum := make([]uint16, 128)

	for i := 0; i < round; i++ {
		index = uint8(matIndex[2*i]) % matNum
		indexnum[index]++
		fmt.Println("round is:", i, "; index is:", index)
		ma = *matrix.Multiply(matList[index], mb)

		index = uint8(matIndex[2*i+1]) % matNum
		indexnum[index]++
		fmt.Println("round is:", i, "; index is:", index)
		mb = *matrix.Multiply(ma, matList[index])
	}

	// mb.Print()

	return &mb
}

func hashMatrix(m *matrix.Matrix) *bc.Hash {
	sha256 := makeHasher(sha3.New256())
	var item []byte
	for i := 1; i <= matSize; i++ {
		for j := 1; j <= matSize; j++ {
			item = append(item, byte(m.Get(i, j)))
		}
	}
	hash := make([]byte, 32)
	sha256(hash, item)
	var arrHash [32]byte
	for i, v := range hash {
		arrHash[i] = v
	}
	newhash := bc.NewHash(arrHash)
	return &newhash
}

var location = []int{
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
