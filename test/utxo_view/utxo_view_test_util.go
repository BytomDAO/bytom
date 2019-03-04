package utxo_view

import (
	"encoding/hex"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/testutil"
)

const utxoPreFix = "UT:"

func calcUtxoKey(hash *bc.Hash) []byte {
	return []byte(utxoPreFix + hash.String())
}

type tx struct {
	Tx *types.Tx
}

func newTx(t *types.Tx) *tx {
	return &tx{
		Tx: t,
	}
}

func (t *tx) getSourceID(outIndex int) *bc.Hash {
	output := t.Tx.Entries[*t.Tx.OutputID(outIndex)].(*bc.Output)
	return output.Source.Ref
}

func (t *tx) getAmount(outIndex int) uint64 {
	output := t.Tx.Entries[*t.Tx.OutputID(outIndex)].(*bc.Output)
	return output.Source.Value.Amount
}

func (t *tx) getSpentOutputID(index int) bc.Hash {
	input, err := t.Tx.Spend(t.Tx.InputIDs[index])
	if err != nil {
		panic(err)
	}

	return *input.SpentOutputId
}

func (t *tx) OutputHash(outIndex int) *bc.Hash {
	return t.Tx.ResultIds[outIndex]
}

func blockNode(header *bc.BlockHeader) *state.BlockNode {
	h := types.BlockHeader{
		Version:           header.Version,
		Height:            header.Height,
		PreviousBlockHash: *header.PreviousBlockId,
		Timestamp:         header.Timestamp,
		Bits:              header.Bits,
		Nonce:             header.Nonce,
	}
	return &state.BlockNode{
		Parent:    nil,
		Hash:      h.Hash(),
		WorkSum:   difficulty.CalcWork(h.Bits),
		Version:   h.Version,
		Height:    h.Height,
		Timestamp: h.Timestamp,
		Nonce:     h.Nonce,
		Bits:      h.Bits,
	}
}

func mustDecodeHex(str string) []byte {
	data, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return data
}

func coinBaseTx(amount uint64, arbitrary string) *types.Tx {
	return types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte(arbitrary)),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, amount, mustDecodeHex("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
		},
	})
}

var mockTransaction = []*tx{}
var mockBlocks = []*block{}

func toHash(hash string) bc.Hash {
	sourceID := bc.Hash{}
	sourceID.UnmarshalText([]byte(hash))
	return sourceID
}

func toAssetID(assetID string) bc.AssetID {
	asset := bc.AssetID{}
	if err := asset.UnmarshalText([]byte(assetID)); err != nil {
		panic(err)
	}
	return asset
}

type block struct {
	types.Block
}

func init() {
	// 0
	mockTransaction = []*tx{}
	t := &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817414d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 41250000000, 0, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00148c704747e94387fa0b8712b053ed2132d84820ac")),
				types.NewTxOutput(*consensus.BTMAssetID, 41150000000, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 1
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[0].getSourceID(1), *consensus.BTMAssetID, 41150000000, 1, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00148c704747e94387fa0b8712b053ed2132d84820ac")),
				types.NewTxOutput(*consensus.BTMAssetID, 41050000000, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 2
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[1].getSourceID(1), *consensus.BTMAssetID, 41050000000, 1, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00148c704747e94387fa0b8712b053ed2132d84820ac")),
				types.NewTxOutput(*consensus.BTMAssetID, 40950000000, []byte("00144431c4278632c6e35dd2870faa1a4b8e0a275cbc")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 3: 00140b0c5059514c751a80c4e1c94f8ecfe16d80671b -> 0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e
	assetID := toAssetID("5c3b60753fe1f8321298d64ab3881b200fa1d7e56f1b2a2df587233c532c5eb6")
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("453936067da4be89a99bbd78aa8c7eb88cbe92ae0941e1013a58b8d6af65d344"), *consensus.BTMAssetID, 41250000000, 0, []byte("00140b0c5059514c751a80c4e1c94f8ecfe16d80671b")),
				types.NewSpendInput(nil, toHash("50d1c966b3a58f9092a696136a75ceb801ea7da2470784d80ebf3f17a76b8a98"), assetID, 800000000000, 0, []byte("00140b0c5059514c751a80c4e1c94f8ecfe16d80671b")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 41150000000, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00140b0c5059514c751a80c4e1c94f8ecfe16d80671b")),
				types.NewTxOutput(assetID, 700000000000, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
				types.NewTxOutput(assetID, 100000000000, []byte("00140b0c5059514c751a80c4e1c94f8ecfe16d80671b")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 4
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 41250000000, 0, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewSpendInput(nil, toHash("d9a9b64e4f842060a40b15325d9aae61987776f7748e7e6a2887a474e84294ef"), assetID, 600000000000, 0, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
				types.NewTxOutput(*consensus.BTMAssetID, 41150000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewTxOutput(assetID, 600000000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
				types.NewTxOutput(assetID, 400000000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 5: 0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e -> 00142b248deeffe82f9cd94fab43849468e0dfe97806
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 41150000000, 1, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
				types.NewSpendInput(nil, toHash("466e6a9261d7b51f227d6c05b7cd3cc36487cc6f0cfb79c58794021e68d4c877"), assetID, 300000000000, 0, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 41050000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
				types.NewTxOutput(assetID, 200000000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewTxOutput(assetID, 100000000000, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	//6: 00142b248deeffe82f9cd94fab43849468e0dfe97806 -> 0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 41050000000, 2, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewSpendInput(nil, toHash("e5757774fb46287ebda3479e19c8643d2fcdb5de3b1ac84d4020c1971bb3f531"), assetID, 100000000000, 0, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40950000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewTxOutput(assetID, 50000000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
				types.NewTxOutput(assetID, 50000000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 7: 0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce -> 0014e3bb841fb722d1840a959d86e12a174c54a3a6e8
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 40950000000, 3, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40850000000, []byte("0014e3bb841fb722d1840a959d86e12a174c54a3a6e8")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 8: 0014e3bb841fb722d1840a959d86e12a174c54a3a6e8 -> 001449601d4cfb6e7a1b990778497b3c364f66bc17d2
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 40850000000, 4, []byte("0014e3bb841fb722d1840a959d86e12a174c54a3a6e8")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40750000000, []byte("001449601d4cfb6e7a1b990778497b3c364f66bc17d2")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014e3bb841fb722d1840a959d86e12a174c54a3a6e8")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 9: 001449601d4cfb6e7a1b990778497b3c364f66bc17d2 -> 0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 40750000000, 5, []byte("001449601d4cfb6e7a1b990778497b3c364f66bc17d2")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40650000000, []byte("0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("001449601d4cfb6e7a1b990778497b3c364f66bc17d2")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 10: 0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b -> 0014e809cb6f328db1e624821dec508cbe08fe1ed08d
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, toHash("ca9b179e549406aa583869e124e39817514d4500a8ce5476e95b6018d182b966"), *consensus.BTMAssetID, 40650000000, 6, []byte("0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40550000000, []byte("0014e809cb6f328db1e624821dec508cbe08fe1ed08d")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// Chain trading
	// 11: 0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e -> 00142b248deeffe82f9cd94fab43849468e0dfe97806
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[3].getSourceID(0), *consensus.BTMAssetID, 41150000000, 0, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 41050000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014b103d8f2dc10e7bbbe2557ff8b9876524dec0a7e")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	//12: 00142b248deeffe82f9cd94fab43849468e0dfe97806 -> 0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[11].getSourceID(0), *consensus.BTMAssetID, 41050000000, 0, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40950000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("00142b248deeffe82f9cd94fab43849468e0dfe97806")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 13: 0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce -> 0014e3bb841fb722d1840a959d86e12a174c54a3a6e8
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[12].getSourceID(0), *consensus.BTMAssetID, 40950000000, 0, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40850000000, []byte("0014e3bb841fb722d1840a959d86e12a174c54a3a6e8")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014492d5b0f09f83bd9bff6a44514dcc9b11c091dce")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 14: 0014e3bb841fb722d1840a959d86e12a174c54a3a6e8 -> 001449601d4cfb6e7a1b990778497b3c364f66bc17d2
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[13].getSourceID(0), *consensus.BTMAssetID, 40850000000, 0, []byte("0014e3bb841fb722d1840a959d86e12a174c54a3a6e8")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40750000000, []byte("001449601d4cfb6e7a1b990778497b3c364f66bc17d2")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014e3bb841fb722d1840a959d86e12a174c54a3a6e8")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 15: 001449601d4cfb6e7a1b990778497b3c364f66bc17d2 -> 0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[14].getSourceID(0), *consensus.BTMAssetID, 40750000000, 0, []byte("001449601d4cfb6e7a1b990778497b3c364f66bc17d2")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40650000000, []byte("0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("001449601d4cfb6e7a1b990778497b3c364f66bc17d2")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	// 16: 0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b -> 0014e809cb6f328db1e624821dec508cbe08fe1ed08d
	t = &tx{
		Tx: types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, *mockTransaction[15].getSourceID(0), *consensus.BTMAssetID, 40650000000, 0, []byte("0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b")),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(*consensus.BTMAssetID, 40550000000, []byte("0014e809cb6f328db1e624821dec508cbe08fe1ed08d")),
				types.NewTxOutput(*consensus.BTMAssetID, 100000000, []byte("0014bd3d70b1bcd62ece61c06a2fe097a4732e5f006b")),
			},
		}),
	}
	mockTransaction = append(mockTransaction, t)

	mockBlocks = []*block{
		// coinbase tx
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            100,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block0"),
			},
		}},

		// Chain trading 3
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            101,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block1"),
				mockTransaction[0].Tx,
				mockTransaction[1].Tx,
				mockTransaction[2].Tx,
			},
		}},

		// detach block 1, attach block 2
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            102,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block2"),
				mockTransaction[0].Tx,
			},
		}},

		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            102,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block3"),
				mockTransaction[0].Tx,
			},
		}},

		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            103,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block4"),
				mockTransaction[1].Tx,
			},
		}},

		// detach block 5, attach block 2
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            104,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block5"),
				mockTransaction[2].Tx,
			},
		}},
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            105,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block6"),
				mockTransaction[3].Tx,
				mockTransaction[4].Tx,
			},
		}},
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            106,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block7"),
				mockTransaction[5].Tx,
			},
		}},
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            107,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block8"),
				mockTransaction[6].Tx,
				mockTransaction[7].Tx,
				mockTransaction[8].Tx,
			},
		}},
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            108,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block9"),
				mockTransaction[9].Tx,
			},
		}},

		// detach block 5, attach block 2. Chain trading
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            105,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block10"),
				mockTransaction[2].Tx,
				mockTransaction[3].Tx,
				mockTransaction[4].Tx,
				mockTransaction[5].Tx,
				mockTransaction[6].Tx,
			},
		}},

		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            105,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block11"),
				mockTransaction[7].Tx,
				mockTransaction[8].Tx,
				mockTransaction[9].Tx,
			},
		}},

		// detach block 2, attach block 1. Chain trading
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            106,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block12"),
				mockTransaction[11].Tx,
				mockTransaction[12].Tx,
				mockTransaction[13].Tx,
				mockTransaction[14].Tx,
			},
		}},
		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            107,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block13"),
				mockTransaction[15].Tx,
				mockTransaction[16].Tx,
			},
		}},

		&block{Block: types.Block{
			BlockHeader: types.BlockHeader{
				Height:            106,
				PreviousBlockHash: testutil.MustDecodeHash("0ab29c0bd7bff3b3b7eb98802f8d5f8833884c86c0fb21559a65cc58dda99667"),
				Timestamp:         1522908275,
				Nonce:             0,
			},
			Transactions: []*types.Tx{
				coinBaseTx(41250000000, "arbitrary block14"),
			},
		}},
	}

}
