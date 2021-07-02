package initblocks

import (
	"fmt"
	"github.com/bytom/bytom/protocol/bc/types"
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

		block := new(types.Block)
		if err := block.UnmarshalText(text); err != nil {
			t.Fatal(err)
		}
	}
}
