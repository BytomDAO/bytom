package blockchain

import (
	"fmt"
	"encoding/json"

	"github.com/bytom/mining"
	"github.com/bytom/protocol/bc/legacy"
)

// Get the parameters of mining
func (bcr *BlockchainReactor) getWork() Response {
	var resp WorkResp
	if block, err := mining.NewBlockTemplate(bcr.chain, bcr.txPool, bcr.accounts); err != nil {
		return NewErrorResponse(err)
	} else {
		resp.Header = block.BlockHeader
	}
	seedCaches := bcr.chain.SeedCaches()
	if seedCache, err := seedCaches.Get(&resp.Header.Seed); err != nil {
		return NewErrorResponse(err)
	} else {
		fmt.Printf("----seed cashe:%v\n", seedCache)
		resp.Cache = seedCache
	}
	if res, err := resp.MarshalJSON(); err == nil {
		fmt.Printf("---------res:%v\n", res)
		var test WorkResp
		err = test.UnmarshalJSON(res)
		fmt.Printf("----------test:%v, err:%v\n", test, err)
		return NewSuccessResponse(res)
	} else {
		return NewErrorResponse(err)
	}
}

type WorkResp struct {
	Header legacy.BlockHeader
	Cache  []uint32
}

type WorkByte struct {
	ByteHeader []byte `json:"header"`
	Cache	   []uint32 `json:"cache"`
}

func (work *WorkResp) UnmarshalJSON(b []byte) error {
	var workByte WorkByte
	if err := json.Unmarshal(b, &workByte); err != nil {
		return err
	}

	if err := work.Header.UnmarshalText(workByte.ByteHeader); err != nil {
		return err
	}
	work.Cache = workByte.Cache

	return nil
}

func (work *WorkResp) MarshalJSON() ([]byte, error) {
	var workByte WorkByte
	var err error
	workByte.ByteHeader, err = work.Header.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(workByte)
}
