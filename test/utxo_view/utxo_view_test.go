package utxo_view

import (
	"os"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/testutil"
)

func TestAttachOrDetachBlocks(t *testing.T) {
	cases := []struct {
		desc        string
		before      map[bc.Hash]*storage.UtxoEntry
		want        map[bc.Hash]*storage.UtxoEntry
		attachBlock []*bc.Block
		detachBlock []*bc.Block
	}{
		{
			desc:   "coinbase tx",
			before: make(map[bc.Hash]*storage.UtxoEntry),
			want:   map[bc.Hash]*storage.UtxoEntry{*newTx(mockBlocks[0].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[0].Block.Height, false)},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[0].Block),
			},
		},
		{
			desc: "Chain trading 3",
			before: map[bc.Hash]*storage.UtxoEntry{
				newTx(mockBlocks[1].Transactions[1]).getSpentOutputID(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[1].Height-1, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[1].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[1].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[1].Block),
			},
		},
		{
			desc: "detach 1 block, attach 2 block",
			before: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[2].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[2].Height, false),
				*newTx(mockBlocks[2].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[2].Height, false),
				*newTx(mockBlocks[2].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[2].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[3].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[3].Height, false),
				*newTx(mockBlocks[3].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[3].Height, false),

				*newTx(mockBlocks[4].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[4].Height, false),
				*newTx(mockBlocks[4].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[4].Height, false),
				*newTx(mockBlocks[4].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[4].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[3].Block),
				types.MapBlock(&mockBlocks[4].Block),
			},
			detachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[2].Block),
			},
		},
		{
			desc: "detach block 5, attach block 2",
			before: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[5].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[5].Height, false),
				*newTx(mockBlocks[5].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[5].Height, false),
				*newTx(mockBlocks[5].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[5].Height, false),

				*newTx(mockBlocks[6].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[6].Height, false),

				*newTx(mockBlocks[7].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[7].Height, false),

				*newTx(mockBlocks[8].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[8].Height, false),

				*newTx(mockBlocks[9].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[9].Height, false),
				*newTx(mockBlocks[9].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[9].Height, false),
				*newTx(mockBlocks[9].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[9].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[10].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(2): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(3): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[10].Height, false),

				*newTx(mockBlocks[11].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[11].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[10].Block),
				types.MapBlock(&mockBlocks[11].Block),
			},
			detachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[9].Block),
				types.MapBlock(&mockBlocks[8].Block),
				types.MapBlock(&mockBlocks[7].Block),
				types.MapBlock(&mockBlocks[6].Block),
				types.MapBlock(&mockBlocks[5].Block),
			},
		},
		{
			desc: "detach block 2, attach block 1. Chain trading",
			before: map[bc.Hash]*storage.UtxoEntry{
				// coinbase tx
				*newTx(mockBlocks[12].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[4]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[12].Height, false),

				*newTx(mockBlocks[13].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[13].Height, false),
				*newTx(mockBlocks[13].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[13].Height, false),
				*newTx(mockBlocks[13].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[13].Height, false),
				*newTx(mockBlocks[13].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(storage.NormalUTXOType, mockBlocks[13].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				newTx(mockBlocks[12].Transactions[1]).getSpentOutputID(0): storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				*newTx(mockBlocks[14].Transactions[0]).OutputHash(0):      storage.NewUtxoEntry(storage.CoinbaseUTXOType, mockBlocks[14].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[14].Block),
			},
			detachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[13].Block),
				types.MapBlock(&mockBlocks[12].Block),
			},
		},
	}
	mockBlockHeader := &mockBlocks[0].Block.BlockHeader
	defer os.RemoveAll("temp")

	for index, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		store := database.NewStore(testDB)

		utxoViewpoint := state.NewUtxoViewpoint()
		for k, v := range c.before {
			utxoViewpoint.Entries[k] = v
		}
		contractView := state.NewContractViewpoint()
		if err := store.SaveChainStatus(mockBlockHeader, []*types.BlockHeader{mockBlockHeader}, utxoViewpoint, contractView, 0, &bc.Hash{}); err != nil {
			t.Error(err)
		}

		utxoViewpoint = state.NewUtxoViewpoint()
		for _, block := range c.detachBlock {
			if err := store.GetTransactionsUtxo(utxoViewpoint, block.Transactions); err != nil {
				t.Error(err)
			}
			if err := utxoViewpoint.DetachBlock(block); err != nil {
				t.Error(err)
			}
		}

		for _, block := range c.attachBlock {
			if err := store.GetTransactionsUtxo(utxoViewpoint, block.Transactions); err != nil {
				t.Error(err)
			}
			if err := utxoViewpoint.ApplyBlock(block); err != nil {
				t.Error(err)
			}
		}
		if err := store.SaveChainStatus(mockBlockHeader, []*types.BlockHeader{mockBlockHeader}, utxoViewpoint, contractView, 0, &bc.Hash{}); err != nil {
			t.Error(err)
		}

		want := map[string]*storage.UtxoEntry{}
		result := make(map[string]*storage.UtxoEntry)

		for k, v := range c.want {
			want[string(database.CalcUtxoKey(&k))] = v
		}

		iter := testDB.IteratorPrefix([]byte(database.UtxoKeyPrefix))
		defer iter.Release()

		for iter.Next() {
			utxoEntry := &storage.UtxoEntry{}
			if err := proto.Unmarshal(iter.Value(), utxoEntry); err != nil {
				t.Error(err)
			}
			key := string(iter.Key())
			result[key] = utxoEntry
		}

		if !testutil.DeepEqual(want, result) {
			t.Errorf("case [%d] fail. want: %v, result: %v", index, want, result)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}
