package utxo_view

import (
	"os"
	"testing"

	"github.com/bytom/testutil"

	"github.com/golang/protobuf/proto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/database/leveldb"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

func TestAttachOrDetachBlocks(t *testing.T) {
	cases := []struct {
		desc           string
		before         map[bc.Hash]*storage.UtxoEntry
		want           map[bc.Hash]*storage.UtxoEntry
		attachBlock    []*bc.Block
		detachBlock    []*bc.Block
		attachTxStatus []*bc.TransactionStatus
		detachTxStatus []*bc.TransactionStatus
	}{
		{
			desc:   "coinbase tx",
			before: make(map[bc.Hash]*storage.UtxoEntry),
			want:   map[bc.Hash]*storage.UtxoEntry{*newTx(mockBlocks[0].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[0].Block.Height, false)},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[0].Block),
			},
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
		},
		{
			desc: "Chain trading 3",
			before: map[bc.Hash]*storage.UtxoEntry{
				newTx(mockBlocks[1].Transactions[1]).getSpentOutputID(): storage.NewUtxoEntry(false, mockBlocks[1].Height-1, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[1].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[1].Height, false),
				*newTx(mockBlocks[1].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[1].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[1].Block),
			},
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
		},
	}
	node := blockNode(types.MapBlock(&mockBlocks[0].Block).BlockHeader)
	defer os.RemoveAll("temp")
	for index, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		store := leveldb.NewStore(testDB)

		utxoViewpoint := state.NewUtxoViewpoint()
		for k, v := range c.before {
			utxoViewpoint.Entries[k] = v
		}
		if err := store.SaveChainStatus(node, utxoViewpoint); err != nil {
			t.Error(err)
		}

		utxoViewpoint = state.NewUtxoViewpoint()
		for index, block := range c.detachBlock {
			if err := store.GetTransactionsUtxo(utxoViewpoint, block.Transactions); err != nil {
				t.Error(err)
			}
			if err := utxoViewpoint.DetachBlock(block, c.detachTxStatus[index]); err != nil {
				t.Error(err)
			}
		}

		for index, block := range c.attachBlock {
			if err := store.GetTransactionsUtxo(utxoViewpoint, block.Transactions); err != nil {
				t.Error(err)
			}
			if err := utxoViewpoint.ApplyBlock(block, c.attachTxStatus[index]); err != nil {
				t.Error(err)
			}
		}
		if err := store.SaveChainStatus(node, utxoViewpoint); err != nil {
			t.Error(err)
		}

		want := map[string]*storage.UtxoEntry{}
		result := make(map[string]*storage.UtxoEntry)

		for k, v := range c.want {
			want[string(calcUtxoKey(&k))] = v
		}

		iter := testDB.IteratorPrefix([]byte(utxoPreFix))
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
