package legacy

import (
	"bytes"
	"testing"

	"github.com/bytom/protocol/bc"
	"github.com/davecgh/go-spew/spew"
)

func TestMapTx(t *testing.T) {
	// sample data copied from transaction_test.go
	// TODO(bobg): factor out into reusable test utility

	oldTx := sampleTx()
	oldOuts := oldTx.Outputs

	_, header, entryMap := mapTx(oldTx)
	t.Log(spew.Sdump(entryMap))

	if header.Version != 1 {
		t.Errorf("header.Version is %d, expected 1", header.Version)
	}
	if header.SerializedSize != oldTx.SerializedSize {
		t.Errorf("header.SerializedSize is %d, expected %d", header.SerializedSize, oldTx.SerializedSize)
	}
	if len(header.ResultIds) != len(oldOuts) {
		t.Errorf("header.ResultIds contains %d item(s), expected %d", len(header.ResultIds), len(oldOuts))
	}

	for i, oldOut := range oldOuts {
		if resultEntry, ok := entryMap[*header.ResultIds[i]]; ok {
			if newOut, ok := resultEntry.(*bc.Output); ok {
				if *newOut.Source.Value != oldOut.AssetAmount {
					t.Errorf("header.ResultIds[%d].(*output).Source is %v, expected %v", i, newOut.Source.Value, oldOut.AssetAmount)
				}
				if newOut.ControlProgram.VmVersion != 1 {
					t.Errorf("header.ResultIds[%d].(*output).ControlProgram.VMVersion is %d, expected 1", i, newOut.ControlProgram.VmVersion)
				}
				if !bytes.Equal(newOut.ControlProgram.Code, oldOut.ControlProgram) {
					t.Errorf("header.ResultIds[%d].(*output).ControlProgram.Code is %x, expected %x", i, newOut.ControlProgram.Code, oldOut.ControlProgram)
				}
				if !newOut.ExtHash.IsZero() {
					t.Errorf("header.ResultIds[%d].(*output).ExtHash is %x, expected zero", i, newOut.ExtHash.Bytes())
				}
			} else {
				t.Errorf("header.ResultIds[%d] has type %T, expected *Output", i, resultEntry)
			}
		} else {
			t.Errorf("entryMap contains nothing for header.ResultIds[%d] (%x)", i, header.ResultIds[i].Bytes())
		}
	}
}

func TestMapCoinbaseTx(t *testing.T) {
	// define the BTM asset id, the soul asset of Bytom
	var BTMAssetID = &bc.AssetID{
		V0: uint64(18446744073709551615),
		V1: uint64(18446744073709551615),
		V2: uint64(18446744073709551615),
		V3: uint64(18446744073709551615),
	}
	oldTx := &TxData{
		Version: 1,
		Inputs: []*TxInput{
			NewCoinbaseInput(nil),
		},
		Outputs: []*TxOutput{
			NewTxOutput(*BTMAssetID, 800000000000, []byte{1}),
		},
	}
	oldOut := oldTx.Outputs[0]

	_, header, entryMap := mapTx(oldTx)
	t.Log(spew.Sdump(entryMap))

	outEntry, ok := entryMap[*header.ResultIds[0]]
	if !ok {
		t.Errorf("entryMap contains nothing for output")
		return
	}
	newOut, ok := outEntry.(*bc.Output)
	if !ok {
		t.Errorf("header.ResultIds[0] has type %T, expected *Output", outEntry)
		return
	}
	if *newOut.Source.Value != oldOut.AssetAmount {
		t.Errorf("(*output).Source is %v, expected %v", newOut.Source.Value, oldOut.AssetAmount)
		return
	}

	muxEntry, ok := entryMap[*newOut.Source.Ref]
	if !ok {
		t.Errorf("entryMap contains nothing for mux")
		return
	}
	mux, ok := muxEntry.(*bc.Mux)
	if !ok {
		t.Errorf("muxEntry has type %T, expected *Mux", muxEntry)
		return
	}
	if *mux.WitnessDestinations[0].Value != oldOut.AssetAmount {
		t.Errorf("(*Mux).Source is %v, expected %v", newOut.Source.Value, oldOut.AssetAmount)
		return
	}

	if coinbaseEntry, ok := entryMap[*mux.Sources[0].Ref]; ok {
		if coinbase, ok := coinbaseEntry.(*bc.Coinbase); ok {
			if *coinbase.WitnessDestination.Value != oldOut.AssetAmount {
				t.Errorf("(*Coinbase).Source is %v, expected %v", newOut.Source.Value, oldOut.AssetAmount)
			}
		} else {
			t.Errorf("inputEntry has type %T, expected *Coinbase", coinbaseEntry)
		}
	} else {
		t.Errorf("entryMap contains nothing for input")
	}
}
