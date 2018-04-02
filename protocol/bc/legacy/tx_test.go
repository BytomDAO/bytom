package legacy

import (
	"testing"

	"github.com/bytom/protocol/bc"

	"github.com/davecgh/go-spew/spew"
)

func TestTxHashes(t *testing.T) {
	cases := []struct {
		txdata *TxData
		hash   bc.Hash
	}{
		{
			txdata: &TxData{},
			hash:   mustDecodeHash("221ed254e69ffaefc0c52baff161392001f8c3ce5c120e4d2cba26eef0bf9500"),
		},
		{
			txdata: sampleTx(),
			hash:   mustDecodeHash("0fd4368e2348690af748eb1b4d5a661dd7ee2cf3b6555ce788aea391d9c10718"), // todo: verify this value,
		},
	}

	for i, c := range cases {
		txEntries := MapTx(c.txdata)
		if len(txEntries.InputIDs) != len(c.txdata.Inputs) {
			t.Errorf("case %d: len(txEntries.InputIDs) = %d, want %d", i, len(txEntries.InputIDs), len(c.txdata.Inputs))
		}
		if c.hash != txEntries.ID {
			t.Errorf("case %d: got txid %x, want %x. txEntries is:\n%s", i, txEntries.ID.Bytes(), c.hash.Bytes(), spew.Sdump(txEntries))
		}
	}
}

func BenchmarkHashEmptyTx(b *testing.B) {
	tx := &TxData{}
	for i := 0; i < b.N; i++ {
		_ = MapTx(tx)
	}
}

func BenchmarkHashNonemptyTx(b *testing.B) {
	tx := sampleTx()
	for i := 0; i < b.N; i++ {
		_ = MapTx(tx)
	}
}

func sampleTx() *TxData {
	initialBlockHash := mustDecodeHash("03deff1d4319d67baa10a6d26c1fea9c3e8d30e33474efee1a610a9bb49d758d")
	assetID := bc.ComputeAssetID([]byte{1}, &initialBlockHash, 1, &bc.EmptyStringHash)
	return &TxData{
		Version:        1,
		SerializedSize: 66,
		Inputs: []*TxInput{
			NewSpendInput(nil, mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), assetID, 1000000000000, 1, []byte{1}, bc.Hash{}, []byte("input")),
			NewSpendInput(nil, bc.NewHash([32]byte{0x11}), assetID, 1, 1, []byte{2}, bc.Hash{}, []byte("input2")),
		},
		Outputs: []*TxOutput{
			NewTxOutput(assetID, 600000000000, []byte{1}, nil),
			NewTxOutput(assetID, 400000000000, []byte{2}, nil),
		},
		ReferenceData: []byte("distribution"),
	}
}
