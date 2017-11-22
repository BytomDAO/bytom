package main

import (
	"fmt"
	// "reflect"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/consensus/aihash"
	"github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/bc/legacy"
)

const (
	epochLength = 128
)

func main() {
	start := time.Now()

	header := bc.NewHash(sha3.Sum256(nil))

	var height uint64 = 1

	EmptyStringHash_2 := bc.NewHash(sha3.Sum256(nil))
	preSeed := &EmptyStringHash_2

	var preEpochBlockHash []*bc.Hash
	for i := 0; i < epochLength; i++ {
		EmptyStringHash := bc.NewHash(sha3.Sum256(nil))
		preEpochBlockHash = append(preEpochBlockHash, &EmptyStringHash)
	}

	seed := aihash.CreateSeed(preSeed, preEpochBlockHash)
	cache := aihash.CreateCache(seed)
	result := aihash.AIHash(cache, height, &header)

	fmt.Println("\nresult is:", (*result).Bytes())

	end := time.Now()
	delta := end.Sub(start)
	fmt.Println("\n-----------------------------------------------")
	fmt.Printf("functions took this amount of time: %s\n", delta)

}
