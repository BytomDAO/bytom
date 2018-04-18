package tensority

import (
	"reflect"
	"unsafe"
	"runtime"
	"sync"

	"gonum.org/v1/gonum/mat"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
)

const (
	matSize     = 1 << 8 // Size of matrix
	matNum      = 1 << 8 // Number of matrix
)

func mulMatrix(headerhash []byte, cache []uint32) []uint8 {
	ui32data := make([]uint32, matNum*matSize*matSize/4)
	for i := 0; i < 128; i++ {
		start := i * 1024 * 32
		for j := 0; j < 512; j++ {
			copy(ui32data[start+j*32:start+j*32+32], cache[start+j*64:start+j*64+32])
			copy(ui32data[start+512*32+j*32:start+512*32+j*32+32], cache[start+j*64+32:start+j*64+64])
		}
	}

	// Convert our destination slice to a int8 buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&ui32data))
	header.Len *= 4
	header.Cap *= 4
	i8data := *(*[]int8)(unsafe.Pointer(&header))

	f64data := make([]float64, matNum*matSize*matSize)
	for i := 0; i < matNum*matSize*matSize; i++ {
		f64data[i] = float64(i8data[i])
	}

	dataIdentity := make([]float64, matSize*matSize)
	for i := 0; i < 256; i++ {
		dataIdentity[i*257] = float64(1)
	}

	var tmp [matSize][matSize]float64
	var maArr [4][matSize][matSize]float64

	runtime.GOMAXPROCS(4)
	var wg sync.WaitGroup
	wg.Add(4)

	for k := 0; k < 4; k++ {
		go func(i int) {
			defer wg.Done()

			ma := mat.NewDense(matSize, matSize, dataIdentity)
			mc := mat.NewDense(matSize, matSize, make([]float64, matSize*matSize))

			var sequence [32]byte
			sha3pool.Sum256(sequence[:], headerhash[i*8:(i+1)*8])

			for j := 0; j < 2; j++ {
				for k := 0; k < 32; k++ {
					index := int(sequence[k])
					mb := mat.NewDense(matSize, matSize, f64data[index*matSize*matSize:(index+1)*matSize*matSize])
					mc.Mul(ma, mb.T())

					for row := 0; row < matSize; row++ {
						for col := 0; col < matSize; col++ {
							i32v := int32(mc.At(row, col))
							i8v := int8((i32v & 0xff) +
								((i32v >> 8) & 0xff))
							mc.Set(row, col, float64(i8v))
						}
					}
					ma = mc
				}
			}

			for row := 0; row < matSize; row++ {
				for col := 0; col < matSize; col++ {
					maArr[i][row][col] = ma.At(row, col)
				}
			}
		}(k)
	}
	wg.Wait()

	for i := 0; i < 4; i++ {
		for row := 0; row < matSize; row++ {
			for col := 0; col < matSize; col++ {
				i32vtmp := int32(tmp[row][col])
				i32vma := int32(maArr[i][row][col])
				i8v := int8(int32(i32vtmp+i32vma) & 0xff)
				tmp[row][col] = float64(i8v)
			}
		}
	}

	result := make([]uint8, 0)
	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize; j++ {
			result = append(result, uint8(tmp[i][j]))
		}
	}
	return result
}

// hashMatrix hash result of mulMatrix
func hashMatrix(result []uint8) *bc.Hash {
	var mat8 [matSize][matSize]uint8
	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize; j++ {
			mat8[i][j] = result[i*matSize+j]
		}
	}

	var mat32 [matSize][matSize / 4]uint32

	for i := 0; i < matSize; i++ {
		for j := 0; j < matSize/4; j++ {
			mat32[i][j] = ((uint32(mat8[i][j+192])) << 24) |
				((uint32(mat8[i][j+128])) << 16) |
				((uint32(mat8[i][j+64])) << 8) |
				((uint32(mat8[i][j])) << 0)
		}
	}

	for k := matSize; k > 1; k = k / 2 {
		for j := 0; j < k/2; j++ {
			for i := 0; i < matSize/4; i++ {
				mat32[j][i] = fnv(mat32[j][i], mat32[j+k/2][i])
			}
		}
	}

	ui32data := make([]uint32, 0)
	for i := 0; i < matSize/4; i++ {
		ui32data = append(ui32data, mat32[0][i])
	}

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&ui32data))
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
