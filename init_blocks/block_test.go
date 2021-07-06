package initblocks

import (
	"fmt"
	"github.com/bytom/bytom/protocol/bc/types"
	"testing"
)

var file = "/Users/cuihaoxin/go/src/github.com/bytom/bytom/init_blocks/bytom_address_asset_balances.txt"
var blocksFile = "/Users/cuihaoxin/go/src/github.com/bytom/bytom/init_blocks/blocks.txt"

func TestInitBlocks(t *testing.T) {
	asset2distributions, err := mapAssetDistributions(file)
	if err != nil {
		t.Fatal(err)
	}

	blocks := initBlocks(asset2distributions)
	storeBlocks(blocks)
}

func storeBlocks(blocks []*types.Block) error {
	block := blocks[0]
	text, err := block.MarshalText()
	if err != nil {
		return err
	}

	return WriteFile(blocksFile, string(text))
}

func TestAA2(t *testing.T) {
	text, err := ReadWholeFile(blocksFile)
	if err != nil {
		t.Fatal(err)
	}

	block := new(types.Block)
	if err := block.UnmarshalText([]byte(text)); err != nil {
		t.Fatal(err)
	}

	hash := block.Hash()
	fmt.Println("block hash:", hash.String())
}
