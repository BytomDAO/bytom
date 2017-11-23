package main

import (
	"fmt"
	// "reflect"
	"time"

	// log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/consensus/aihash"
	// "github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

const (
	epochLength = 128
)

func main() {
	start := time.Now()

	newbchash := bc.NewHash(sha3.Sum256(nil))
	header := &newbchash

	var height uint64 = 1

	EmptyStringHash_2 := bc.NewHash(sha3.Sum256(nil))
	preSeed := &EmptyStringHash_2

	var preEpochBlockHash []*bc.Hash
	for i := 0; i < epochLength; i++ {
		EmptyStringHash := bc.NewHash(sha3.Sum256(nil))
		preEpochBlockHash = append(preEpochBlockHash, &EmptyStringHash)
	}

	seed := aihash.CreateSeed(height, preSeed, preEpochBlockHash)
	// seed = nil
	cache, err := aihash.CreateCache(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	// header = nil
	result, err := aihash.AIHash(height, header, cache)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("\nresult is:", (*result).Bytes())

	end := time.Now()
	delta := end.Sub(start)
	fmt.Println("\n-----------------------------------------------")
	fmt.Printf("functions took this amount of time: %s\n", delta)
}
