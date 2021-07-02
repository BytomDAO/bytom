package initblocks

import (
	"fmt"
	"github.com/bytom/bytom/protocol/bc/types"
	"testing"
)

var file = "/Users/cuihaoxin/go/src/github.com/bytom/bytom/init_blocks/bytom_address_asset_balances.txt"
var blocksFile = "/Users/cuihaoxin/go/src/github.com/bytom/bytom/init_blocks/blocks.txt"

func TestInitBlocks(t *testing.T) {
	asset2distributions, err := classifyByAsset(file)
	if err != nil {
		t.Fatal(err)
	}

	blocks := initBlocks(asset2distributions)
	fmt.Println("len blocks:", len(blocks))
	block := blocks[0]
	text, err := block.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	hash := block.Hash()
	fmt.Println("block hash:", hash.String())
	WriteFile(blocksFile, string(text))
}

func TestAA2(t *testing.T) {
	//lines, err := ReadFileLines(blocksFile)
	// 没有读进来
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

/*
want:
block hash: 9fee9ba633ce7f1ac1f8da2dfd111aa8af61e10919c65120f55d9a440d8e4ec2
*/

/* unmarshal to block
block hash: 9fee9ba633ce7f1ac1f8da2dfd111aa8af61e10919c65120f55d9a440d8e4ec2
*/
