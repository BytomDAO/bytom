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
			want:   map[bc.Hash]*storage.UtxoEntry{*mockTransaction[0].OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[0].Block.Height, false)},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[0].Block),
			},
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: true},
				}},
			},
		},
		{
			desc: "Chain trading 3",
			before: map[bc.Hash]*storage.UtxoEntry{
				newTx(mockBlocks[1].Transactions[1]).getSpentOutputID(): storage.NewUtxoEntry(false, mockBlocks[1].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[1].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, 101, false),
				*newTx(mockBlocks[1].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, 101, false),
				*newTx(mockBlocks[1].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, 101, false),
				*newTx(mockBlocks[1].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, 101, false),
				*newTx(mockBlocks[1].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, 101, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[1].Block),
			},
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: true},
					&bc.TxVerifyResult{StatusFail: true},
					&bc.TxVerifyResult{StatusFail: true},
					&bc.TxVerifyResult{StatusFail: true},
				}},
			},
		},
	}
	node := blockNode(types.MapBlock(&mockBlocks[0].Block).BlockHeader)
	for _, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		defer os.RemoveAll("temp")
		store := leveldb.NewStore(testDB)

		want := map[string]*storage.UtxoEntry{}
		result := make(map[string]*storage.UtxoEntry)
		for k, v := range c.before {
			utxoViewpoint := state.NewUtxoViewpoint()
			utxoViewpoint.Entries[k] = v
			store.SaveChainStatus(node, utxoViewpoint)
		}

		for index, block := range c.detachBlock {
			utxoViewpoint := state.NewUtxoViewpoint()
			store.GetTransactionsUtxo(utxoViewpoint, block.Transactions)
			utxoViewpoint.ApplyBlock(block, c.detachTxStatus[index])
			store.SaveChainStatus(node, utxoViewpoint)
		}

		for index, block := range c.attachBlock {
			utxoViewpoint := state.NewUtxoViewpoint()
			store.GetTransactionsUtxo(utxoViewpoint, block.Transactions)
			utxoViewpoint.ApplyBlock(block, c.attachTxStatus[index])
			store.SaveChainStatus(node, utxoViewpoint)
		}

		for k, v := range c.want {
			want[string(calcUtxoKey(&k))] = v
		}

		iter := testDB.IteratorPrefix([]byte(utxoPreFix))
		defer iter.Release()

		for iter.Next() {
			utxoEntry := storage.UtxoEntry{}
			if err := proto.Unmarshal(iter.Value(), &utxoEntry); err != nil {
				t.Error(err)
			}
			key := string(iter.Key())
			result[key] = &utxoEntry
		}

		if !testutil.DeepEqual(want, result) {
			t.Error(want)
			t.Error(result)
		}
		testDB.Close()
	}
}
