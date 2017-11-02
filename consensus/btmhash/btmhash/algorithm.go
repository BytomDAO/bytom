package btmhash

import (
	"encoding/binary"
	"fmt"
	"hash"
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/ethereum/go-ethereum/common/bitutil"
	"golang.org/x/crypto/sha3"

	"btmhash/matrix"
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

// seedHash is the seed to use for generating a verification cache and the mining
// dataset.
func seedHash(block uint64) []byte {
	seed := make([]byte, 32)
	if block < epochLength {
		return seed
	}
	sha256 := makeHasher(sha3.New256())
	for i := 0; i < int(block/epochLength); i++ {
		sha256(seed, seed)
	}
	return seed
}

// block contain all the block header data.
// 现在先产生0序列为区块号的种子。等并入主链的时候建议将每个窗口期的区块哈希值合并，
// 然后将前一个种子合并，对结合体进行哈希操作，从而得到当前窗口期的种子
func createSeed(block []byte) []byte {
	seed := make([]byte, 32)
	sha256 := makeHasher(sha3.New256())
	sha256(seed, block)
	return seed
}

// generateCache creates a verification cache of a given size for an input seed.
// The cache production process involves first sequentially filling up 32 MB of
// memory, then performing two passes of Sergio Demian Lerner's RandMemoHash
// algorithm from Strict Memory Hard Hashing Functions (2014). The output is a
// set of 524288 64-byte values.
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

// 从cache中选择数据填充到矩阵中，选择数据的方式
func fillMatrixList(matList []matrix.Matrix, cache []uint32, blockNum uint64) {
	//var block uint64 = 0
	// var epoch uint64 = block / epochLength
	var locationIndex uint64 = blockNum % epochLength

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&cache))
	header.Len *= 4
	header.Cap *= 4
	cacheInt8 := *(*[]int8)(unsafe.Pointer(&header))

	//fmt.Println("cacheInt8 is:", cacheInt8)

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

	fmt.Println("Print the matrix!")

	// for i := 0; i < matNum; i++ {
	// 	fmt.Println("No.", i)
	// 	matList[i].Print()
	// }

	fmt.Println("fill the matrix list!")
	fmt.Println("block number is:", blockNum)
	fmt.Println("locationIndex is:", locationIndex)
	fmt.Println("location[block % epochLength] is:", location[blockNum%epochLength])
}

func getIndex(blockHeader []byte) []byte {
	sha256 := makeHasher(sha3.New256())
	matIndex := make([]byte, 32)
	sha256(matIndex, blockHeader)

	return matIndex
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
	fmt.Println("--------------------------------------------------")
	fmt.Println("initial mb is:")
	// mb.Print()

	fmt.Println("matIndex is:", matIndex)

	//
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

	fmt.Println("----------------------------------------\nindexnum is:", indexnum)

	return &mb
}

var location = []int{
	3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59,
	61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137,
	139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199, 211, 223, 227,
	229, 233, 239, 241, 251, 257, 263, 269, 271, 277, 281, 283, 293, 307, 311, 313,
	317, 331, 337, 347, 349, 353, 359, 367, 373, 379, 383, 389, 397, 401, 409, 419,
	421, 431, 433, 439, 443, 449, 457, 461, 463, 467, 479, 487, 491, 499, 503, 509,
	523, 541, 571, 601, 613, 619, 643, 661, 691, 709, 739, 751, 769, 823, 829, 859,
	991, 1021, 1033, 1069, 1129, 1153, 1171, 1213, 1231, 1321, 1399, 1423, 1453, 1459, 1483, 1489}
