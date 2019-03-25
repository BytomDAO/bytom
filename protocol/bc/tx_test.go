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
			wantHash: "17dfad182df66212f6f694d774285e5989c5d9d1add6d5ce51a5930dbef360d8",
		},
		{
			tx: &Tx{
				ID: Hash{V0: 17091584763764411831, V1: 2315724244669489432, V2: 4322938623810388342, V3: 11167378497724951792},
				InputIDs: []Hash{
					{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
				},
			},
			wantHash: "f650ba3a58f90d3a2215f6c50a692a86c621b7968bb2a059a4c8e0c819770430",
		},
	}

	for _, c := range cases {
		gotHash := c.tx.SigHash(0)
		if gotHash.String() != c.wantHash {
			t.Errorf("got hash:%s, want hash:%s", gotHash.String(), c.wantHash)
		}
	}
}
