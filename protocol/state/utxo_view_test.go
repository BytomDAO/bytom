package state

import (
	"testing"

	"github.com/bytom/blockchain/txdb/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestApplyBlock(t *testing.T) {
	cases := []struct {
		block     *bc.Block
		inputView *UtxoViewpoint
		fetchView *UtxoViewpoint
		err       bool
	}{
		{
			block: &bc.Block{
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: NewUtxoViewpoint(),
			err:       true,
		},
		{
			block: &bc.Block{
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
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
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
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
					Height: 7,
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
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
					Height: 0,
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
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
		if err := c.inputView.ApplyBlock(c.block); c.err != (err != nil) {
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
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			err:       true,
		},
		{
			block: &bc.Block{
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
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
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
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
		if err := c.inputView.DetachBlock(c.block); c.err != (err != nil) {
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
