package main

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol/bc"
)

func main() {
	fmt.Println("test tensority!")

	var b32 [32]byte
	seed128 := extendBytes(b32[:], 3)
	hash := bc.NewHash(b32)
	seed := bc.NewHash(b32)

	fmt.Println("hash: ", hex.EncodeToString(hash.Bytes()[:]))

	fmt.Println("seed(128B):")
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			fmt.Printf("0x")
			for k := 0; k < 4; k++ {
				fmt.Printf(hex.EncodeToString(seed128[i*16+j*4+k : i*16+j*4+k+1]))
			}
			fmt.Printf(", ")
		}
		fmt.Printf("\n")
	}
	fmt.Println("seed128(hex): ", hex.EncodeToString(seed128))

	start := time.Now()
	result := tensority.Hash(&hash, &seed).Bytes()
	end := time.Now()
	fmt.Println("time: ", end.Sub(start))
	fmt.Println("result: ", result)
	fmt.Println("result(hex): ", hex.EncodeToString(result))
}

func extendBytes(seed []byte, round int) []byte {
	extSeed := make([]byte, len(seed)*(round+1))
	copy(extSeed, seed)

	for i := 0; i < round; i++ {
		// h := crypto.Sha256(extSeed[i*32 : (i+1)*32])
		var h [32]byte
		sha3pool.Sum256(h[:], extSeed[i*32:(i+1)*32])
		copy(extSeed[(i+1)*32:(i+2)*32], h[:])
	}

	return extSeed
}
