package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/testutil"
)

func TestReadWriteSpendCommitment(t *testing.T) {
	btmAssetID := testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	cases := []struct {
		sc           *SpendCommitment
		encodeString string
	}{
		{
			sc: &SpendCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &btmAssetID,
					Amount:  100,
				},
				SourceID:       testutil.MustDecodeHash("3160fb24f97e06ad5a9717cd47fe2b65c7409903216b39120b10550282b20e99"),
				SourcePosition: 0,
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("0014d927424f4e8c242460b538f04c2676b97842e9a7"),
				StateData:      [][]byte{testutil.MustDecodeHexString("1234abcd")},
			},
			encodeString: "603160fb24f97e06ad5a9717cd47fe2b65c7409903216b39120b10550282b20e99ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff640001160014d927424f4e8c242460b538f04c2676b97842e9a701041234abcd",
		},
		{
			sc: &SpendCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &btmAssetID,
					Amount:  999,
				},
				SourceID:       testutil.MustDecodeHash("4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adea"),
				SourcePosition: 2,
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("001418549d84daf53344d32563830c7cf979dc19d5c0"),
				StateData:      [][]byte{testutil.MustDecodeHexString("123456abcdef")},
			},
			encodeString: "634b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adeaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe707020116001418549d84daf53344d32563830c7cf979dc19d5c00106123456abcdef",
		},
	}

	for _, c := range cases {
		buff := []byte{}
		buffer := bytes.NewBuffer(buff)
		if err := c.sc.writeExtensibleString(buffer, nil, 1); err != nil {
			t.Fatal(err)
		}

		got := hex.EncodeToString(buffer.Bytes())
		if got != c.encodeString {
			t.Errorf("test write spend commitment fail, got:%s, want:%s", got, c.encodeString)
		}

		sc := &SpendCommitment{}
		_, err := sc.readFrom(blockchain.NewReader(buffer.Bytes()), 1)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(*sc, *c.sc) {
			t.Errorf("test read spend commitment fail, got:%v, want:%v", *sc, *c.sc)
		}
	}
}
