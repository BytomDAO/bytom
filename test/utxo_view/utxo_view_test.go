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
				newTx(mockBlocks[1].Transactions[1]).getSpentOutputID(0): storage.NewUtxoEntry(false, mockBlocks[1].Height-1, false),
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
		{
			desc: "detach 1 block, attach 2 block",
			before: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[2].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[2].Height, false),
				*newTx(mockBlocks[2].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[2].Height, false),
				*newTx(mockBlocks[2].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[2].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[3].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[3].Height, false),
				*newTx(mockBlocks[3].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[3].Height, false),

				*newTx(mockBlocks[4].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[4].Height, false),
				*newTx(mockBlocks[4].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[4].Height, false),
				*newTx(mockBlocks[4].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[4].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[3].Block),
				types.MapBlock(&mockBlocks[4].Block),
			},
			detachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[2].Block),
			},
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
			detachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
		},
		{
			desc: "detach block 5, attach block 2",
			before: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[5].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[5].Height, false),
				*newTx(mockBlocks[5].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[5].Height, false),
				*newTx(mockBlocks[5].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[5].Height, false),

				*newTx(mockBlocks[6].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[6].Height, false),

				*newTx(mockBlocks[7].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[7].Height, false),

				*newTx(mockBlocks[8].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),

				*newTx(mockBlocks[9].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[9].Height, false),
				*newTx(mockBlocks[9].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[9].Height, false),
				*newTx(mockBlocks[9].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[9].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[10].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),

				*newTx(mockBlocks[11].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
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
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
			detachTxStatus: []*bc.TransactionStatus{

				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
		},
		{
			desc: "detach block 5, attach block 2. Other asset deals failed.",
			before: map[bc.Hash]*storage.UtxoEntry{
				*newTx(mockBlocks[5].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[5].Height, false),
				*newTx(mockBlocks[5].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[5].Height, false),
				*newTx(mockBlocks[5].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[5].Height, false),

				*newTx(mockBlocks[6].Transactions[0]).OutputHash(0):      storage.NewUtxoEntry(true, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(0):      storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[1]).OutputHash(1):      storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				newTx(mockBlocks[6].Transactions[1]).getSpentOutputID(1): storage.NewUtxoEntry(false, mockBlocks[6].Height-1, false),

				*newTx(mockBlocks[6].Transactions[2]).OutputHash(0):      storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				*newTx(mockBlocks[6].Transactions[2]).OutputHash(1):      storage.NewUtxoEntry(false, mockBlocks[6].Height, false),
				newTx(mockBlocks[6].Transactions[2]).getSpentOutputID(1): storage.NewUtxoEntry(false, mockBlocks[6].Height-1, false),

				*newTx(mockBlocks[7].Transactions[0]).OutputHash(0):      storage.NewUtxoEntry(true, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(0):      storage.NewUtxoEntry(false, mockBlocks[7].Height, false),
				*newTx(mockBlocks[7].Transactions[1]).OutputHash(1):      storage.NewUtxoEntry(false, mockBlocks[7].Height, false),
				newTx(mockBlocks[7].Transactions[1]).getSpentOutputID(1): storage.NewUtxoEntry(false, mockBlocks[7].Height-1, false),

				*newTx(mockBlocks[8].Transactions[0]).OutputHash(0):      storage.NewUtxoEntry(true, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(0):      storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[1]).OutputHash(1):      storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				newTx(mockBlocks[8].Transactions[1]).getSpentOutputID(1): storage.NewUtxoEntry(false, mockBlocks[8].Height-1, false),

				*newTx(mockBlocks[8].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),
				*newTx(mockBlocks[8].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[8].Height, false),

				*newTx(mockBlocks[9].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[9].Height, false),
				*newTx(mockBlocks[9].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[9].Height, false),
				*newTx(mockBlocks[9].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[9].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{

				*newTx(mockBlocks[10].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),

				*newTx(mockBlocks[10].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[2]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),

				*newTx(mockBlocks[10].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[3]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),

				*newTx(mockBlocks[10].Transactions[4]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[4]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(2): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),
				*newTx(mockBlocks[10].Transactions[5]).OutputHash(3): storage.NewUtxoEntry(false, mockBlocks[10].Height, false),

				*newTx(mockBlocks[11].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[1]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[3]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
				*newTx(mockBlocks[11].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[11].Height, false),
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
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
			detachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: true},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: true},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: true},
					&bc.TxVerifyResult{StatusFail: true},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
		},
		{
			desc: "detach block 2, attach block 1. Chain trading",
			before: map[bc.Hash]*storage.UtxoEntry{
				// coinbase tx
				*newTx(mockBlocks[12].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[3]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[12].Height, false),
				*newTx(mockBlocks[12].Transactions[4]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[12].Height, false),

				*newTx(mockBlocks[13].Transactions[0]).OutputHash(0): storage.NewUtxoEntry(true, mockBlocks[13].Height, false),
				*newTx(mockBlocks[13].Transactions[1]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[13].Height, false),
				*newTx(mockBlocks[13].Transactions[2]).OutputHash(0): storage.NewUtxoEntry(false, mockBlocks[13].Height, false),
				*newTx(mockBlocks[13].Transactions[2]).OutputHash(1): storage.NewUtxoEntry(false, mockBlocks[13].Height, false),
			},
			want: map[bc.Hash]*storage.UtxoEntry{
				newTx(mockBlocks[12].Transactions[1]).getSpentOutputID(0): storage.NewUtxoEntry(false, 0, false),
				*newTx(mockBlocks[14].Transactions[0]).OutputHash(0):      storage.NewUtxoEntry(true, mockBlocks[14].Height, false),
			},
			attachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[14].Block),
			},
			detachBlock: []*bc.Block{
				types.MapBlock(&mockBlocks[13].Block),
				types.MapBlock(&mockBlocks[12].Block),
			},
			attachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
			},
			detachTxStatus: []*bc.TransactionStatus{
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
					&bc.TxVerifyResult{StatusFail: false},
				}},
				&bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{
					&bc.TxVerifyResult{StatusFail: false},
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
