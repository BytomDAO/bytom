package main

import (
	"fmt"

	"github.com/bytom/mining/tensority"
	"github.com/bytom/protocol/bc"
)

func main() {
	fmt.Println("test tensority!")

	b32 := make([]byte, 32)
	hash := bc.NewHash(b32)
	seed := bc.NewHash()

	fmt.Println(tensority.Hash())
}
