package state

import (
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
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

var gasOnlyTxEntry = map[bc.Hash]bc.Entry{
	bc.Hash{V1: 0}: &bc.Output{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: consensus.BTMAssetID,
			},
		},
	},
	bc.Hash{V1: 1}: &bc.Output{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: &bc.AssetID{V0: 999},
			},
		},
	},
}

func TestApplyBlock(t *testing.T) {
	cases := []struct {
		block     *bc.Block
		inputView *UtxoViewpoint
		fetchView *UtxoViewpoint
		gasOnlyTx bool
		err       bool
	}{
		{
			// can't find prevout in tx entries
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 1},
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
					Height:            101,
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
		{
			// output will be store
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V0: 0},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        defaultEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(true, 0, false),
				},
			},
			err: false,
		},
		{
			// apply gas only tx, non-btm asset spent input will not be spent
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
							bc.Hash{V1: 0},
							bc.Hash{V1: 1},
						},
						Entries: gasOnlyTxEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(false, 0, false),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(false, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(false, 0, true),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(false, 0, false),
				},
			},
			gasOnlyTx: true,
			err:       false,
		},
		{
			// apply gas only tx, non-btm asset spent output will not be store
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V1: 0},
								&bc.Hash{V1: 1},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        gasOnlyTxEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(true, 0, false),
				},
			},
			gasOnlyTx: true,
			err:       false,
		},
	}

	for i, c := range cases {
		c.block.TransactionStatus.SetStatus(0, c.gasOnlyTx)
		if err := c.inputView.ApplyBlock(c.block, c.block.TransactionStatus); c.err != (err != nil) {
			t.Errorf("case #%d want err = %v, get err = %v", i, c.err, err)
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
		gasOnlyTx bool
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
							ResultIds: []*bc.Hash{
								&bc.Hash{V0: 0},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        defaultEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
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
							bc.Hash{V1: 0},
							bc.Hash{V1: 1},
						},
						Entries: gasOnlyTxEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(false, 0, true),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(false, 0, true),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(false, 0, false),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(false, 0, true),
				},
			},
			gasOnlyTx: true,
			err:       false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V1: 0},
								&bc.Hash{V1: 1},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        gasOnlyTxEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(false, 0, true),
				},
			},
			gasOnlyTx: true,
			err:       false,
		},
	}

	for i, c := range cases {
		c.block.TransactionStatus.SetStatus(0, c.gasOnlyTx)
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
