package types

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
)

func TestMapSpendTx(t *testing.T) {
	cases := []*TxData{
		&TxData{
			Inputs: []*TxInput{
				NewSpendInput(nil, testutil.MustDecodeHash("fad5195a0c8e3b590b86a3c0a95e7529565888508aecca96e9aeda633002f409"), *consensus.BTMAssetID, 88, 3, []byte{1}),
			},
			Outputs: []*TxOutput{
				NewTxOutput(*consensus.BTMAssetID, 80, []byte{1}),
			},
		},
		&TxData{
			Inputs: []*TxInput{
				NewIssuanceInput([]byte("nonce"), 254354, []byte("issuanceProgram"), [][]byte{[]byte("arguments1"), []byte("arguments2")}, []byte("assetDefinition")),
			},
			Outputs: []*TxOutput{
				NewTxOutput(*consensus.BTMAssetID, 80, []byte{1}),
			},
		},
		&TxData{
			Inputs: []*TxInput{
				NewIssuanceInput([]byte("nonce"), 254354, []byte("issuanceProgram"), [][]byte{[]byte("arguments1"), []byte("arguments2")}, []byte("assetDefinition")),
				NewSpendInput(nil, testutil.MustDecodeHash("db7b16ac737440d6e38559996ddabb207d7ce84fbd6f3bfd2525d234761dc863"), *consensus.BTMAssetID, 88, 3, []byte{1}),
			},
			Outputs: []*TxOutput{
				NewTxOutput(*consensus.BTMAssetID, 80, []byte{1}),
				NewTxOutput(*consensus.BTMAssetID, 80, []byte{1}),
			},
		},
	}

	for _, txData := range cases {
		tx := MapTx(txData)
		if len(tx.ResultIds) != len(txData.Outputs) {
			t.Errorf("ResultIds contains %d item(s), expected %d", len(tx.ResultIds), len(txData.Outputs))
		}

		for i, oldIn := range txData.Inputs {
			resultEntry, ok := tx.Entries[tx.InputIDs[i]]
			if !ok {
				t.Errorf("entryMap contains nothing for tx.InputIDs[%d] (%x)", i, tx.InputIDs[i].Bytes())
			}
			switch newInput := resultEntry.(type) {
			case *bc.Issuance:
				if *newInput.Value.AssetId != oldIn.AssetID() || newInput.Value.Amount != oldIn.Amount() {
					t.Errorf("tx.InputIDs[%d]'s asset amount is not equal after map'", i)
				}
			case *bc.Spend:
				spendOut, err := tx.Output(*newInput.SpentOutputId)
				if err != nil {
					t.Fatal(err)
				}
				if *spendOut.Source.Value != oldIn.AssetAmount() {
					t.Errorf("tx.InputIDs[%d]'s asset amount is not equal after map'", i)
				}
			default:
				t.Errorf("unexpect input type")
			}
		}

		for i, oldOut := range txData.Outputs {
			resultEntry, ok := tx.Entries[*tx.ResultIds[i]]
			if !ok {
				t.Errorf("entryMap contains nothing for header.ResultIds[%d] (%x)", i, tx.ResultIds[i].Bytes())
			}
			newOut, ok := resultEntry.(*bc.Output)
			if !ok {
				t.Errorf("header.ResultIds[%d] has type %T, expected *Output", i, resultEntry)
			}

			if *newOut.Source.Value != oldOut.AssetAmount {
				t.Errorf("header.ResultIds[%d].(*output).Source is %v, expected %v", i, newOut.Source.Value, oldOut.AssetAmount)
			}
			if newOut.ControlProgram.VmVersion != 1 {
				t.Errorf("header.ResultIds[%d].(*output).ControlProgram.VMVersion is %d, expected 1", i, newOut.ControlProgram.VmVersion)
			}
			if !bytes.Equal(newOut.ControlProgram.Code, oldOut.ControlProgram) {
				t.Errorf("header.ResultIds[%d].(*output).ControlProgram.Code is %x, expected %x", i, newOut.ControlProgram.Code, oldOut.ControlProgram)
			}

		}
	}
}

func TestMapCoinbaseTx(t *testing.T) {
	txData := &TxData{
		Inputs: []*TxInput{
			NewCoinbaseInput([]byte("TestMapCoinbaseTx")),
		},
		Outputs: []*TxOutput{
			NewTxOutput(*consensus.BTMAssetID, 800000000000, []byte{1}),
		},
	}
	oldOut := txData.Outputs[0]

	tx := MapTx(txData)
	t.Log(spew.Sdump(tx.Entries))

	if len(tx.InputIDs) != 1 {
		t.Errorf("expect to  only have coinbase input id")
	}
	if len(tx.SpentOutputIDs) != 0 {
		t.Errorf("coinbase tx doesn't spend any utxo")
	}
	if len(tx.GasInputIDs) != 1 {
		t.Errorf("coinbase tx should have 1 gas input")
	}
	if len(tx.ResultIds) != 1 {
		t.Errorf("expect to  only have one output")
	}

	outEntry, ok := tx.Entries[*tx.ResultIds[0]]
	if !ok {
		t.Errorf("entryMap contains nothing for output")
	}
	newOut, ok := outEntry.(*bc.Output)
	if !ok {
		t.Errorf("header.ResultIds[0] has type %T, expected *Output", outEntry)
	}
	if *newOut.Source.Value != oldOut.AssetAmount {
		t.Errorf("(*output).Source is %v, expected %v", newOut.Source.Value, oldOut.AssetAmount)
	}

	muxEntry, ok := tx.Entries[*newOut.Source.Ref]
	if !ok {
		t.Errorf("entryMap contains nothing for mux")
	}
	mux, ok := muxEntry.(*bc.Mux)
	if !ok {
		t.Errorf("muxEntry has type %T, expected *Mux", muxEntry)
	}
	if *mux.WitnessDestinations[0].Value != *newOut.Source.Value {
		t.Errorf("(*Mux).Destinations is %v, expected %v", *mux.WitnessDestinations[0].Value, *newOut.Source.Value)
	}

	coinbaseEntry, ok := tx.Entries[tx.InputIDs[0]]
	if !ok {
		t.Errorf("entryMap contains nothing for coinbase input")
	}
	coinbase, ok := coinbaseEntry.(*bc.Coinbase)
	if !ok {
		t.Errorf("inputEntry has type %T, expected *Coinbase", coinbaseEntry)
	}
	if coinbase.WitnessDestination.Value != mux.Sources[0].Value {
		t.Errorf("(*Coinbase).Destination is %v, expected %v", coinbase.WitnessDestination.Value, *mux.Sources[0].Value)
	}
}
