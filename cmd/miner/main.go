package main

import (
	"fmt"

	"github.com/bytom/util"
	"github.com/bytom/blockchain"
)

func doWork(work *blockchain.WorkResp) {
	fmt.Printf("work:%v", work)
}

func main() {
	data, exitCode := util.ClientCall("/get-work")
	if exitCode != util.Success {
		return
	}
	if work, ok := data.(blockchain.WorkResp); ok {
		doWork(&work)
	}
}
