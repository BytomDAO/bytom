package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/testutil"
)

func TestReadWriteSupLink(t *testing.T) {
	cases := []struct {
		desc      string
		supLinks  SupLinks
		hexString string
	}{
		{
			desc: "normal sup links",
			supLinks: []*SupLink{
				{
					SourceHash: testutil.MustDecodeHash("0a3cd1175e295a35c2b63054969c3fe54eeaa3eb68258227b28d8daa6cf4c50c"),
					Signatures: [consensus.NumOfValidators][]byte{
						testutil.MustDecodeHexString("750318156e8c913c378a8d31294fca1084df3be3967035425f470f81e00cd824d1f12bf8e1c3b308f4b1a916438b9ce630722bc8d92ef0feebbbaf987dd7a60e"),
						testutil.MustDecodeHexString("be7c7e0ba54109c8c457cdbba4691d7aaae32eb4b8ac63755f2494be406027ce66c7b4730bfd2506fa2caaba12a7bbbea2faca5f07bb64fe06a568b6415e7506"),
					},
				},
				{
					SourceHash: testutil.MustDecodeHash("546c91cefc6a06f9b7a0aaa4d69db9a7f229af27928304a44ecd48e33ba2ba91"),
					Signatures: [consensus.NumOfValidators][]byte{
						testutil.MustDecodeHexString("38c9a6a48eeea993b2d4137e73b17e4743ce3935636fcce957ae2291c691491525f39509a1c21fec3c7f78403ae88e375b796fa9dcc4cac0af8a987994f62c07"),
						testutil.MustDecodeHexString("4fe5646b2b669aaef0dd74a090e150de676218d0a6e693bb2d1cc791282517669d7903c60a909a5d9c5a996e5797ea9dded20b52dc4b8ec272e86e5fc4e8a008"),
					},
				},
			},
			hexString: "020a3cd1175e295a35c2b63054969c3fe54eeaa3eb68258227b28d8daa6cf4c50c40750318156e8c913c378a8d31294fca1084df3be3967035425f470f81e00cd824d1f12bf8e1c3b308f4b1a916438b9ce630722bc8d92ef0feebbbaf987dd7a60e40be7c7e0ba54109c8c457cdbba4691d7aaae32eb4b8ac63755f2494be406027ce66c7b4730bfd2506fa2caaba12a7bbbea2faca5f07bb64fe06a568b6415e75060000000000000000546c91cefc6a06f9b7a0aaa4d69db9a7f229af27928304a44ecd48e33ba2ba914038c9a6a48eeea993b2d4137e73b17e4743ce3935636fcce957ae2291c691491525f39509a1c21fec3c7f78403ae88e375b796fa9dcc4cac0af8a987994f62c07404fe5646b2b669aaef0dd74a090e150de676218d0a6e693bb2d1cc791282517669d7903c60a909a5d9c5a996e5797ea9dded20b52dc4b8ec272e86e5fc4e8a0080000000000000000",
		},
		{
			desc:      "empty sup links",
			supLinks:  []*SupLink{},
			hexString: "00",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			buff := []byte{}
			buffer := bytes.NewBuffer(buff)
			if err := c.supLinks.writeTo(buffer); err != nil {
				t.Fatal(err)
			}

			hexString := hex.EncodeToString(buffer.Bytes())
			if hexString != c.hexString {
				t.Errorf("test write suplinks fail, got:%s, want:%s", hexString, c.hexString)
			}

			supLinks := SupLinks{}
			if err := supLinks.readFrom(blockchain.NewReader(buffer.Bytes())); err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(supLinks, c.supLinks) {
				t.Errorf("test read suplinks fail, got:%v, want:%v", supLinks, c.supLinks)
			}
		})
	}
}
