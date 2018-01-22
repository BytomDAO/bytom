package main

import (
	"fmt"

	"github.com/bytom/util"
	"github.com/bytom/blockchain"
//	"github.com/bytom/protocol/bc/legacy"
)

func doWork(work *blockchain.WorkResp) {
	fmt.Printf("work:%v", work)
}

func main() {
	data, exitCode := util.ClientCall("/get-work")
	fmt.Printf("data:%v", data)
	if exitCode != util.Success {
		return
	}
/*	var work blockchain.WorkResp
	if dataMap, ok := data.(map[string]interface{}); ok {
		work.Header = dataMap["header"].(legacy.BlockHeader)
		fmt.Printf("work:%v", work)
		doWork(&work)
	}*/
}
