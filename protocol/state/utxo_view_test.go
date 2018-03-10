package state

import (
	"testing"

	"github.com/bytom/blockchain/txdb/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

var defaultEntry = map[bc.Hash]bc.Entry{
	bc.Hash{V0: 0}: &bc.Output{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: &bc.AssetID{V0: 0},
			},
		},
	},
}

func TestApplyBlock(t *testing.T) {
	cases := []struct {
		block     *bc.Block
		inputView *UtxoViewpoint
		fetchView *UtxoViewpoint
		err       bool
	}{
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: NewUtxoViewpoint(),
			err:       true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, true),
				},
			},
			err: true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, true),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:            7,
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(true, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(true, 0, true),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:            0,
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(true, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(true, 0, true),
				},
			},
			err: true,
		},
	}

	for i, c := range cases {

		if err := c.inputView.ApplyBlock(c.block, c.block.TransactionStatus); c.err != (err != nil) {
			t.Errorf("want err = %v, get err = %v", c.err, err)
		}
		if c.err {
			continue
		}
		if !testutil.DeepEqual(c.inputView, c.fetchView) {
			t.Errorf("test case %d, want %v, get %v", i, c.fetchView, c.inputView)
		}
	}
}

func TestDetachBlock(t *testing.T) {
	cases := []struct {
		block     *bc.Block
		inputView *UtxoViewpoint
		fetchView *UtxoViewpoint
		err       bool
	}{
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, false),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, false),
				},
			},
			err: true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, true),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(false, 0, false),
				},
			},
			err: false,
		},
	}

	for i, c := range cases {
		if err := c.inputView.DetachBlock(c.block, c.block.TransactionStatus); c.err != (err != nil) {
			t.Errorf("case %d want err = %v, get err = %v", i, c.err, err)
		}
		if c.err {
			continue
		}
		if !testutil.DeepEqual(c.inputView, c.fetchView) {
			t.Errorf("test case %d, want %v, get %v", i, c.fetchView, c.inputView)
		}
	}
}
