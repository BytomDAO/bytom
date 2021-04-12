package integration

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/bytom/bytom/protocol/bc"
)

func TestHash(t *testing.T) {
	s := "dcaafb317d6faee190410e0c9b99b8e2ac84e748188e54a48c6569890f83ff38"
	bytes, _ := hex.DecodeString(s)

	var bs [32]byte
	copy(bs[:], bytes)
	h := bc.NewHash(bs)
	fmt.Println("newHash:", h.String())
	fmt.Println("oldHash:", s)
}

func TestPrintBlockMap(t *testing.T) {
	for height := 0; height < 4; height++ {
		blocks := blockMap[height]
		for i, block := range blocks {
			hash := block.block.Hash()
			fmt.Printf("height:%d,index:%d,hash:%s \n", height, i, hash.String())
		}
	}
}
