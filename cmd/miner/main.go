package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/bytom/api"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/util"
)

const (
	maxNonce = ^uint64(0) // 2^64 - 1
	isCrazy = false
)

// do proof of work
func doWork(bh *types.BlockHeader, seed *bc.Hash) bool {
	for i := uint64(0); i <= maxNonce; i++ {
		bh.Nonce = i
		headerHash := bh.Hash()
		if difficulty.CheckProofOfWork(&headerHash, seed, bh.Bits) {
			log.Printf("Mining succeed! Proof hash: %v\n", headerHash.String())
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
		BlockHeader *types.BlockHeader `json:"block_header"`
		Reward      uint64             `json:"reward"`
	}

	data, _ := util.ClientCall("/get-block-header-by-height", Req{BlockHeight: height})
	rawData, err := json.Marshal(data)
	if err != nil {
		log.Fatalln(err)
	}

	resp := &Resp{}
	if err = json.Unmarshal(rawData, resp); err != nil {
		log.Fatalln(err)
	}
	log.Println("Reward:", resp.Reward)
}

func main() {
	for true {
		data, _ := util.ClientCall("/get-work", nil)
		if data == nil {
			os.Exit(1)
		}
		rawData, err := json.Marshal(data)
		if err != nil {
			log.Fatalln(err)
		}
		resp := &api.GetWorkResp{}
		if err = json.Unmarshal(rawData, resp); err != nil {
			log.Fatalln(err)
		}

		log.Println("Mining at height:", resp.BlockHeader.Height)
		if doWork(resp.BlockHeader, resp.Seed) {
			util.ClientCall("/submit-work", &api.SubmitWorkReq{BlockHeader: resp.BlockHeader})
		}

		getBlockHeaderByHeight(resp.BlockHeader.Height)

		if !isCrazy {
			return
		}	
	}
}
