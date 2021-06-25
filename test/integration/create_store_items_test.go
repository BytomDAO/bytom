package integration

import (
	"fmt"
	"testing"

	"github.com/bytom/bytom/database"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/testutil"
)

func TestCreateStoreItems(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	mainChainIndex := []int{0, 1, 2}
	blocks := []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][3]}
	itme := &storeItem{
		key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
		val: &storage.UtxoEntry{Type: storage.NormalUTXOType, BlockHeight: 1, Spent: false},
	}

	items := createStoreItems(mainChainIndex, blocks, itme)
	fmt.Println(items)
}
