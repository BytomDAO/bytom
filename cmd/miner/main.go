package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytom/blockchain"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/util"
)

const (
	maxNonce = ^uint64(0) // 2^32 - 1
)

// do proof of work
func doWork(bh *legacy.BlockHeader, seed *bc.Hash) bool {
	for i := uint64(0); i <= maxNonce; i++ {
		bh.Nonce = i
		headerHash := bh.Hash()
		if difficulty.CheckProofOfWork(&headerHash, seed, bh.Bits) {
			fmt.Printf("Mining: successful-----proof hash:%v\n", headerHash.String())
			return true
		}
	}
	return false
}

func getBlockHeaderByHeight(height uint64) {
	type Req struct {
		BlockHeight uint64 `json:"block_height"`
	}

	type Resp struct {
		BlockHeader *legacy.BlockHeader `json:"block_header"`
		Reward      uint64              `json:"reward"`
	}

	data, _ := util.ClientCall("/get-block-header-by-height", Req{BlockHeight: height})
	rawData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	resp := &Resp{}
	if err = json.Unmarshal(rawData, resp); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(resp.Reward)
}

func main() {
	data, _ := util.ClientCall("/getwork", nil)
	rawData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	resp := &blockchain.GetWorkResp{}
	if err = json.Unmarshal(rawData, resp); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if doWork(resp.BlockHeader, resp.Seed) {
		util.ClientCall("/submitwork", resp.BlockHeader)
	}

	getBlockHeaderByHeight(resp.BlockHeader.Height)
}
