package main

import (
	"context"
	"fmt"

	"github.com/bytom/blockchain"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/util"
)

const (
	maxNonce = ^uint64(0) // 2^32 - 1
)

// do proof of work
func doWork(work *blockchain.WorkResp) {
	fmt.Printf("work:%v\n", work)
	for i := uint64(0); i <= maxNonce; i++ {
		work.Header.Nonce = i
		headerHash := work.Header.Hash()

		if difficulty.CheckProofOfWork(&headerHash, work.Header.Bits) {
			// to do: submitWork
			fmt.Printf("Mining: successful-----proof hash:%v\n", headerHash)
			return
		}
	}
}

func main() {
	var work blockchain.WorkResp
	client := util.MustRPCClient()
	if err := client.Call(context.Background(), "/get-work", nil, &work); err == nil {
		doWork(&work)
	} else {
		fmt.Printf("---err:%v\n", err)
	}
}
