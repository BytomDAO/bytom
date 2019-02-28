package state

import (
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
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

var usedUtxoHash = []bc.Hash{}

func generateSpendTx(tx bc.Tx) *types.Tx {
	for _, v := range tx.TxHeader.ResultIds {
		output := tx.Entries[*v].(*bc.Output)
		if string(output.ControlProgram.Code) == "00144431c4278632c6e35dd2870faa1a4b8e0a275cbc" {
			tx := types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil, *output.Source.Ref, *consensus.BTMAssetID, output.Source.Value.Amount, output.Source.Position, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00148c704747e94387fa0b8712b053ed2132d84820ac")),
					types.NewTxOutput(*consensus.BTMAssetID, output.Source.Value.Amount-100000000, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
				},
			})
			usedUtxoHash = append(usedUtxoHash, *v)
			return tx
		}
	}
	return nil
}

func generateBlock(hash bc.Hash, txs []*bc.Tx, h uint64) *bc.Block {
	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:            h,
			PreviousBlockId:   &hash,
			Timestamp:         1522908275,
			Nonce:             0,
			TransactionStatus: bc.NewTransactionStatus(),
		},
		Transactions: txs,
	}
	return block
}

func generateCoinBaseTx(amount uint64) *types.Tx {
	return types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("arbitrary")),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, amount, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
		},
	})
}

func TestApplyBlocks(t *testing.T) {
	utxoViewpoint := NewUtxoViewpoint()
	// coinbase
	hash := bc.Hash{}
	hash.UnmarshalText([]byte("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"))
	txs := []*bc.Tx{}
	txCoinBase := generateCoinBaseTx(41250000000)
	txs = append(txs, txCoinBase.Tx)

	b := generateBlock(hash, txs, 101)
	b.TransactionStatus.SetStatus(0, true)
	utxoViewpoint.ApplyBlock(b, b.TransactionStatus)

	// spend
	txs = txs[0:0]
	/// coinbase tx
	tx := generateCoinBaseTx(41350000000)
	txs = append(txs, tx.Tx)
	/// spend conbase tx
	tx = generateSpendTx(*txCoinBase.Tx)
	txs = append(txs, tx.Tx)

	/// spend normal tx
	for i := 0; i < 6; i++ {
		tx = generateSpendTx(*tx.Tx)
		txs = append(txs, tx.Tx)
	}

	b = generateBlock(hash, txs, 101+101)

	for i := 0; i < len(txs); i++ {
		b.TransactionStatus.SetStatus(i, true)
	}

	if err := utxoViewpoint.ApplyBlock(b, b.TransactionStatus); err != nil {
		t.Error(err)
	}

	usedUtxoHashNum := len(usedUtxoHash)
	num := 0
	for _, v := range usedUtxoHash {
		utxoEntry, ok := utxoViewpoint.Entries[v]
		if ok && utxoEntry.Spent {
			num++
		}
	}
	if usedUtxoHashNum != num {
		t.Errorf("utxoViewpoint: %v", utxoViewpoint.Entries)
		t.Errorf("usedUtxoHash: %v", usedUtxoHash)
	}
}
