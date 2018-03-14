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
			hash:   mustDecodeHash("202f9971e4b61016d3d771f7a3506c0fc0a850c76c8469ae13e87f58d11fbb1a"),
		},
		{
			txdata: sampleTx(),
			hash:   mustDecodeHash("33c121707a5c5567ed2c0a63da1fbc4077d51bc367b9630c664bfc241b51e641"), // todo: verify this value,
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
			NewSpendInput(nil, mustDecodeHash("dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292"), assetID, 1000000000000, 1, []byte{1}, bc.Hash{}),
			NewSpendInput(nil, bc.NewHash([32]byte{0x11}), assetID, 1, 1, []byte{2}, bc.Hash{}),
		},
		Outputs: []*TxOutput{
			NewTxOutput(assetID, 600000000000, []byte{1}),
			NewTxOutput(assetID, 400000000000, []byte{2}),
		},
	}
}
