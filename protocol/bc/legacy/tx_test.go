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
			hash:   mustDecodeHash("02439cf4a8d801d10e84f5b3818226e38dac889dc626b7a1b5888b49510b38fe"),
		},
		{
			txdata: sampleTx(),
			hash:   mustDecodeHash("360eab1b2563e85d9a3f290f3f2c0d99c622c89088f8c2e2003000fbee62cca0"), // todo: verify this value,
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
