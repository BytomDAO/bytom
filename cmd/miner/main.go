package main

import (
	"context"
	"fmt"

	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/util"
)

const (
	maxNonce = ^uint64(0) // 2^32 - 1
)

// do proof of work
func doWork(bh *legacy.BlockHeader) bool {
	for i := uint64(0); i <= maxNonce; i++ {
		bh.Nonce = i
		headerHash := bh.Hash()
		if difficulty.CheckProofOfWork(&headerHash, bh.Bits) {
			fmt.Printf("Mining: successful-----proof hash:%v\n", headerHash)
			return true
		}
	}
	return false
}

func main() {
	client := util.MustRPCClient()
	for {
		bh := &legacy.BlockHeader{}
		if err := client.Call(context.Background(), "/minepool/get-work", nil, bh); err != nil {
			fmt.Println(err)
			break
		}

		if doWork(bh) {
			client.Call(context.Background(), "/minepool/submit-work", bh, nil)
		}
	}
}
