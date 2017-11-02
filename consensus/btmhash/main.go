package main

import (
	"fmt"
	"time"

	"btmhash/btmhash"
)

func main() {
	start := time.Now()

	btmhash.TestBtmhash()
	//btmhash.TestSHA3()
	//btmhash.TestSeedHash()
	//btmhash.TestGenerateCache()
	//btmhash.TestRandomness()
	//btmhash.Testnonce()

	end := time.Now()
	delta := end.Sub(start)
	fmt.Println("\n-----------------------------------------------")
	fmt.Printf("functions took this amount of time: %s\n", delta)
}
