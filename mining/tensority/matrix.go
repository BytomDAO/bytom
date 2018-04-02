package tensority

import (
	"reflect"
	"unsafe"

	"gonum.org/v1/gonum/mat"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
)

const (
	matSize     = 1 << 8 // Size of matrix
	matNum      = 1 << 8 // Number of matrix
	epochLength = 1 << 7 // Blocks per epoch
	mulRounds   = 1 << 8 // Number of rounds in mulmatrix
)

func mulMatrix(headerhash []byte, cache []uint32) []uint8 {
	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&cache))
	header.Len *= 4
	header.Cap *= 4
	cacheInt8 := *(*[]int8)(unsafe.Pointer(&header))

	data := make([]float64, matNum*matSize*matSize)
	for i := 0; i < len(cacheInt8); i++ {
		data[i] = float64(cacheInt8[i])
	}

	exthash := extendBytes(headerhash, 7)

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

// TODO: check why
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
		for j := 0; j < matSize/4; j++ {
			mat32[i][j] = ((uint32(mat8[i][j+192])) << 24) |
				((uint32(mat8[i][j+128])) << 16) |
				((uint32(mat8[i][j+64])) << 8) |
				((uint32(mat8[i][j])) << 0)
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

	for i := 0; i < matSize/4; i++ {
		data = append(data, mat32[0][i])
	}

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&data))
	header.Len *= 4
	header.Cap *= 4
	dataBytes := *(*[]byte)(unsafe.Pointer(&header))

	var h [32]byte
	sha3pool.Sum256(h[:], dataBytes)
	bcHash := bc.NewHash(h)
	return &bcHash
}

func fnv(a, b uint32) uint32 {
	return a*0x01000193 ^ b
}
