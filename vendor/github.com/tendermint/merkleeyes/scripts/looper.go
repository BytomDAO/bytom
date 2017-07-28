package main

import (
	"fmt"

	. "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/db"
	"github.com/tendermint/merkleeyes/iavl"
)

func main() {
	db := db.NewMemDB()
	t := iavl.NewIAVLTree(0, db)
	// 23000ns/op, 43000ops/s
	// for i := 0; i < 10000000; i++ {
	// for i := 0; i < 1000000; i++ {
	for i := 0; i < 1000; i++ {
		t.Set(RandBytes(12), nil)
	}
	t.Save()

	fmt.Println("ok, starting")

	for i := 0; ; i++ {
		key := RandBytes(12)
		t.Set(key, nil)
		t.Remove(key)
		if i%1000 == 0 {
			t.Save()
		}
	}
}
