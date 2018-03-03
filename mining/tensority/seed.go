package tensority

import (
	"encoding/binary"
	"unsafe"

	"github.com/bytom/crypto/scrypt"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc"
)

func calcSeed(blockHashs []*bc.Hash) []byte {
	data := []byte{}
	for _, blockHash := range blockHashs {
		data = append(data, blockHash.Bytes()...)
	}
	var s [32]byte
	sha3pool.Sum256(s[:], data)

	return s[:]
}

func extendBytes(seed []byte, round int) []byte {
	extSeed := make([]byte, len(seed)*(round+1))
	copy(extSeed, seed)

	for i := 0; i < round; i++ {
		var h [32]byte
		sha3pool.Sum256(h[:], extSeed[i*32:(i+1)*32])
		copy(extSeed[(i+1)*32:(i+2)*32], h[:])
	}

	return extSeed
}

func calcSeedCache(seed []byte) (cache []uint32) {
	extSeed := extendBytes(seed, 3)
	v := make([]uint32, 32*1024)

	// Swap the byte order on big endian systems
	if !isLittleEndian() {
		swap(extSeed)
	}

	for i := 0; i < 128; i++ {
		scrypt.Smix(extSeed, v)
		cache = append(cache, v...)
	}

	return cache
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
