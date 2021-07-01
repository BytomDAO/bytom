package initblocks

import (
	"fmt"
	"testing"
)

func TestInitBlocks(t *testing.T) {
	blocks := initBlocks(assetTotals, asset2distributions)
	for _, block := range blocks {
		text, err := block.MarshalText()
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(string(text))
	}
}
