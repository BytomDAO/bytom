package main

import (
	"fmt"

	"github.com/bytom/blockchain"
	"github.com/bytom/util"
)

func doWork(work *blockchain.WorkResp) {
	fmt.Printf("work:%v\n", work)
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
