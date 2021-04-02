package integration

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/bytom/bytom/consensus/difficulty"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func TestPoW(t *testing.T) {
	block := blockMap[3][1].block
	s := "9e6291970cb44dd94008c79bcaf9d86f18b4b49ba5b2a04781db7199ed3b9e4e"
	SolveBlock(s, block)
}

func SolveBlock(s string, block *types.Block) error {
	bytes, _ := hex.DecodeString(s)
	var bs [32]byte
	copy(bs[:], bytes)
	seed := bc.NewHash(bs)

	maxNonce := ^uint64(0) // 2^64 - 1
	header := &block.BlockHeader
	for i := uint64(0); i < maxNonce; i++ {
		header.Nonce = i
		headerHash := header.Hash()
		if difficulty.CheckProofOfWork(&headerHash, &seed, header.Bits) {
			fmt.Printf("nonce:%v, headerHash:%s \n", header.Nonce, headerHash.String())
			return nil
		}
	}

	return errors.New("not found nonce")
}

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
