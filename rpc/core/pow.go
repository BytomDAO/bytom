package core

import (
        ctypes "github.com/bytom/rpc/core/types"
        "github.com/bytom/protocol/bc/legacy"
)

func GetWork()(*ctypes.ResultBlockHeaderInfo, error){
    return &ctypes.ResultBlockHeaderInfo{},nil
}

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

