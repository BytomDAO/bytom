package main

import (
	"encoding/json"
	"fmt"
	"os"

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
			fmt.Printf("Mining: successful-----proof hash:%v\n", headerHash.String())
			return true
		}
	}
	return false
}

func main() {
	data, _ := util.ClientCall("/getwork", nil)
	rawData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	bh := &legacy.BlockHeader{}
	if err = json.Unmarshal(rawData, bh); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if doWork(bh) {
		util.ClientCall("/submitwork", &bh)
	}

}
