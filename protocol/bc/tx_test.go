package bc

import (
	"testing"
)

func TestSigHash(t *testing.T) {
	cases := []struct {
		tx       *Tx
		wantHash string
	}{
		{
			tx: &Tx{
				ID: Hash{V0: 13464118406972499748, V1: 5083224803004805715, V2: 16263625389659454272, V3: 9428032044180324575},
				InputIDs: []Hash{
					{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
				},
			},
			wantHash: "e350ff74b8fa278ff78ce74d7776a71f3dcd25b7c1c7955c6c448ec87b80959e",
		},
		{
			tx: &Tx{
				ID: Hash{V0: 17091584763764411831, V1: 2315724244669489432, V2: 4322938623810388342, V3: 11167378497724951792},
				InputIDs: []Hash{
					{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
				},
			},
			wantHash: "ffb7ed58d00530f2e1032fc90a04318aead9b87e36c635b0002743015ae8f95d",
		},
	}

	for _, c := range cases {
		gotHash := c.tx.SigHash(0)
		if gotHash.String() != c.wantHash {
			t.Errorf("got hash:%s, want hash:%s", gotHash.String(), c.wantHash)
		}
	}
}
