package core

import (
        //"fmt"
        //"context"
        //"github.com/blockchain/protocol/bc"
        ctypes "github.com/bytom/rpc/core/types"
        //"github.com/blockchain/protocol"
        //"chain/protocol/bc/legacy" 
        //"github.com/consensus/types"
        //. "github.com/tendermint/tmlibs/common"sour	
        "github.com/bytom/protocol/bc/legacy"
)

//for simulate
func GetWork()(*ctypes.ResultBlockHeaderInfo, error){
    //ctx := context.Background()
    //b1 := &legacy.Block{BlockHeader: legacy.BlockHeader{Height: 0}}
    return &ctypes.ResultBlockHeaderInfo{},nil
}


//func SubmitWork(blkheader ctypes.ResultBlockHeaderInfo) (bool,error) {
func SubmitWork(height uint64) (bool,error) {
    block := legacy.Block{
                BlockHeader: legacy.BlockHeader{
                    Version: 1,
                    Height: height,
                },
    }
    blockStore.SaveBlock(&block)
    return true,nil
}

