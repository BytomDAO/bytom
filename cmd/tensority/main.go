package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bytom/common/hexutil"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol/bc"
)

func main() {
	fmt.Println("test tensority!")

	var b32hash [32]byte
	var b32seed [32]byte

	sbhash := hexutil.MustDecode("0xd0dad73fb2dabf3353fda15571b4e5f6ac62ff187b354fadd4840d9ff2f1afdf")
	sbseed := hexutil.MustDecode("0x0737520781345b11b7bd0f843c1bdd9aea81b6da94fd141cc9f2df53ac6744d2")

	copy(b32hash[:], sbhash)
	copy(b32seed[:], sbseed)

	b128seed := extendBytes(b32seed[:], 3)
	fmt.Println("b128seed: ", hex.EncodeToString(b128seed))
	swap(b128seed)
	hash := bc.NewHash(b32hash)
	seed := bc.NewHash(b32seed)

	fmt.Println("hash: ", hex.EncodeToString(hash.Bytes()[:]))
	for i := 0; i < 4; i++ {
		for j := 0; j < 8; j++ {
			fmt.Printf("0x%v, ", hex.EncodeToString(sbhash[i*8+j:i*8+j+1]))
		}
		fmt.Println("")
	}

	fmt.Println("seed(128B):")
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			fmt.Printf("0x")
			for k := 0; k < 4; k++ {
				fmt.Printf(hex.EncodeToString(b128seed[i*16+j*4+k : i*16+j*4+k+1]))
			}
			fmt.Printf(", ")
		}
		fmt.Printf("\n")
	}
	fmt.Println("seed128(hex): ", hex.EncodeToString(b128seed))

	start := time.Now()
	result := tensority.Hash(&hash, &seed).Bytes()
	end := time.Now()
	fmt.Println("time: ", end.Sub(start))
	// fmt.Println("result: ", result)
	fmt.Println("result: ", hex.EncodeToString(result))
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

// swap changes the byte order of the buffer assuming a uint32 representation.
func swap(buffer []byte) {
	for i := 0; i < len(buffer); i += 4 {
		binary.BigEndian.PutUint32(buffer[i:], binary.LittleEndian.Uint32(buffer[i:]))
	}
}
