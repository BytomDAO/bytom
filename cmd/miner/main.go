package main

import (
	"fmt"

	"github.com/bytom/blockchain"
	"github.com/bytom/consensus/algorithm"
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
		proofHash, err := algorithm.AIHash(work.Header.Height, &headerHash, work.Cache)
		if err != nil {
			fmt.Printf("Mining: failed on AIHash: %v\n", err)
			return
		}

		if difficulty.CheckProofOfWork(proofHash, work.Header.Bits) {
			// to do: submitWork
			fmt.Printf("Mining: successful-----proof hash:%v\n", proofHash)
			return
		}
	}
}

func main() {
	data, exitCode := util.ClientCall("/get-work")
	fmt.Printf("data:%v\n", data)
	if exitCode != util.Success {
		return
	}

	var work blockchain.WorkResp
	if err := work.UnmarshalJSON([]byte(data.(string))); err == nil {
		doWork(&work)
	} else {
		fmt.Printf("---err:%v\n", err)
	}
}
