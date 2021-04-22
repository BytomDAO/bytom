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
						nil,
						nil,
						testutil.MustDecodeHexString("4dd9508652a686b37247d2fa969ca92997095cec44aa2ceb223daf29c1c426f5e06d3e522e85161386ad70af2c04e703179e749870f6e669b0038067338fe709"),
						testutil.MustDecodeHexString("3ab09481823ee2caff6939ea0e70693d63173c4295975be6bbf030554941de2babfb66fc3c795f026785fdf2f5090617f05292816d0ccb83f8d2dc487e3ad404"),
						nil,
						testutil.MustDecodeHexString("9e48c3852c16189dd82b48c43de6460771802caab373dc8e572c0e510edcc6341e7b070dec6a2068d2519e044eaadc609ae6c3233cdcbb713ef0546edfa2f803"),
						testutil.MustDecodeHexString("b26b8f5fb33b800b8d06768304864138b0ece5ce7e57fcc339f714911d279d103b08a5f8a85c1723dfe0299690ad776fb8b11e003ddfc33749b5000d0a78350f"),
					},
				},
			},
			hexString: "020a3cd1175e295a35c2b63054969c3fe54eeaa3eb68258227b28d8daa6cf4c50c40750318156e8c913c378a8d31294fca1084df3be3967035425f470f81e00cd824d1f12bf8e1c3b308f4b1a916438b9ce630722bc8d92ef0feebbbaf987dd7a60e40be7c7e0ba54109c8c457cdbba4691d7aaae32eb4b8ac63755f2494be406027ce66c7b4730bfd2506fa2caaba12a7bbbea2faca5f07bb64fe06a568b6415e75060000000000000000546c91cefc6a06f9b7a0aaa4d69db9a7f229af27928304a44ecd48e33ba2ba914038c9a6a48eeea993b2d4137e73b17e4743ce3935636fcce957ae2291c691491525f39509a1c21fec3c7f78403ae88e375b796fa9dcc4cac0af8a987994f62c07404fe5646b2b669aaef0dd74a090e150de676218d0a6e693bb2d1cc791282517669d7903c60a909a5d9c5a996e5797ea9dded20b52dc4b8ec272e86e5fc4e8a0080000404dd9508652a686b37247d2fa969ca92997095cec44aa2ceb223daf29c1c426f5e06d3e522e85161386ad70af2c04e703179e749870f6e669b0038067338fe709403ab09481823ee2caff6939ea0e70693d63173c4295975be6bbf030554941de2babfb66fc3c795f026785fdf2f5090617f05292816d0ccb83f8d2dc487e3ad40400409e48c3852c16189dd82b48c43de6460771802caab373dc8e572c0e510edcc6341e7b070dec6a2068d2519e044eaadc609ae6c3233cdcbb713ef0546edfa2f80340b26b8f5fb33b800b8d06768304864138b0ece5ce7e57fcc339f714911d279d103b08a5f8a85c1723dfe0299690ad776fb8b11e003ddfc33749b5000d0a78350f00",
		},
		{
			desc: "sup links with full signature",
			supLinks: []*SupLink{
				{
					SourceHash: testutil.MustDecodeHash("0a3cd1175e295a35c2b63054969c3fe54eeaa3eb68258227b28d8daa6cf4c50c"),
					Signatures: [consensus.NumOfValidators][]byte{
						testutil.MustDecodeHexString("750318156e8c913c378a8d31294fca1084df3be3967035425f470f81e00cd824d1f12bf8e1c3b308f4b1a916438b9ce630722bc8d92ef0feebbbaf987dd7a60e"),
						testutil.MustDecodeHexString("be7c7e0ba54109c8c457cdbba4691d7aaae32eb4b8ac63755f2494be406027ce66c7b4730bfd2506fa2caaba12a7bbbea2faca5f07bb64fe06a568b6415e7506"),
						testutil.MustDecodeHexString("9938ea16d6caae68b7e9318f1aed387ef9767dc0d80db807e0d0a77065229ceffef7a8b6407882f5d6e29b2edf1c6373bb1c47188138068e2baa4851c04c6f0e"),
						testutil.MustDecodeHexString("4dd9508652a686b37247d2fa969ca92997095cec44aa2ceb223daf29c1c426f5e06d3e522e85161386ad70af2c04e703179e749870f6e669b0038067338fe709"),
						testutil.MustDecodeHexString("3ab09481823ee2caff6939ea0e70693d63173c4295975be6bbf030554941de2babfb66fc3c795f026785fdf2f5090617f05292816d0ccb83f8d2dc487e3ad404"),
						testutil.MustDecodeHexString("52a13c4502265fb456f8ecd051de7b6059b5ad59a741ed561efc06489f161b0d471d86f3bf62ef0083e603a26b98abc945018b8f94f591782d43deb5df1dec08"),
						testutil.MustDecodeHexString("9e48c3852c16189dd82b48c43de6460771802caab373dc8e572c0e510edcc6341e7b070dec6a2068d2519e044eaadc609ae6c3233cdcbb713ef0546edfa2f803"),
						testutil.MustDecodeHexString("4103f5e7939f1e83241580251a56d85f31cedbca0be7ea819e352ab61aebdb047419e2775704539af4897bdd65e0cf69dc7e82b9e338efe88b5e7eb911dd8303"),
						testutil.MustDecodeHexString("b26b8f5fb33b800b8d06768304864138b0ece5ce7e57fcc339f714911d279d103b08a5f8a85c1723dfe0299690ad776fb8b11e003ddfc33749b5000d0a78350f"),
						testutil.MustDecodeHexString("30a9b6922a04ad7e72310842d589da14edfc3a81d60e3d6d934bd4adff4c3bb78a8506fcbe1323a21d2058a294c4af7a5a961e4e033380e2ed150ef0dcfbcb00"),
					},
				},
			},
			hexString: "010a3cd1175e295a35c2b63054969c3fe54eeaa3eb68258227b28d8daa6cf4c50c40750318156e8c913c378a8d31294fca1084df3be3967035425f470f81e00cd824d1f12bf8e1c3b308f4b1a916438b9ce630722bc8d92ef0feebbbaf987dd7a60e40be7c7e0ba54109c8c457cdbba4691d7aaae32eb4b8ac63755f2494be406027ce66c7b4730bfd2506fa2caaba12a7bbbea2faca5f07bb64fe06a568b6415e7506409938ea16d6caae68b7e9318f1aed387ef9767dc0d80db807e0d0a77065229ceffef7a8b6407882f5d6e29b2edf1c6373bb1c47188138068e2baa4851c04c6f0e404dd9508652a686b37247d2fa969ca92997095cec44aa2ceb223daf29c1c426f5e06d3e522e85161386ad70af2c04e703179e749870f6e669b0038067338fe709403ab09481823ee2caff6939ea0e70693d63173c4295975be6bbf030554941de2babfb66fc3c795f026785fdf2f5090617f05292816d0ccb83f8d2dc487e3ad4044052a13c4502265fb456f8ecd051de7b6059b5ad59a741ed561efc06489f161b0d471d86f3bf62ef0083e603a26b98abc945018b8f94f591782d43deb5df1dec08409e48c3852c16189dd82b48c43de6460771802caab373dc8e572c0e510edcc6341e7b070dec6a2068d2519e044eaadc609ae6c3233cdcbb713ef0546edfa2f803404103f5e7939f1e83241580251a56d85f31cedbca0be7ea819e352ab61aebdb047419e2775704539af4897bdd65e0cf69dc7e82b9e338efe88b5e7eb911dd830340b26b8f5fb33b800b8d06768304864138b0ece5ce7e57fcc339f714911d279d103b08a5f8a85c1723dfe0299690ad776fb8b11e003ddfc33749b5000d0a78350f4030a9b6922a04ad7e72310842d589da14edfc3a81d60e3d6d934bd4adff4c3bb78a8506fcbe1323a21d2058a294c4af7a5a961e4e033380e2ed150ef0dcfbcb00",
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
