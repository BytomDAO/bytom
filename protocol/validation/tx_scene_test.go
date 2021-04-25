package validation

import (
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/vm"
)

func TestValidateTx(t *testing.T) {
	converter := func(prog []byte) ([]byte, error) { return nil, nil }
	cases := []struct {
		desc   string
		txData *types.TxData
		err    error
	}{
		{
			desc: "single utxo, single sign, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 331,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("556a2be7ea4e116e6ff9a0df0ababc2541fce9e8a0f209b624e1cb1f55e8f2f1f3ed4097b4b2daa12bab8f7746ef5c7966788e9d89daf08e11c8de78d115460d"),
							mustDecodeString("32fa23244a69e5524d190ad62391c2fc654685a740e00e9a316b78c95028363f"),
						},
						bc.Hash{V0: 2},
						*consensus.BTMAssetID, 1000000000, 0, mustDecodeString("00149dd32abe4756676cc310470457edefce8b3bd7e7"), []byte{}),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 100, mustDecodeString("00149dd32abe4756676cc310470457edefce8b3bd7e7"), []byte{}),
				},
			},
			err: nil,
		},
		{
			desc: "multi utxo, single sign, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 595,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("bc68d6cf8e9b58ad1561fe855a2e2072941833ccc73efb2db82181a275133711087975e7f16e5004bf5f4214a99c2326e77c8ee9005112251209f799c50a3e06"),
							mustDecodeString("3c4518eb4faa8ab01503057933932f33503e7693e05bb20948efd525be5850df"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014a5f3f1941449c6072ade0ec5a66fad1417124f03"), nil),
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("634d17cf09126ec42547d878414a78703accbc52b98d44d3bd71bb78a035cc1663f5340ecee915290c8c89c552a9d1795bf0dee7832cedafb7be15d0ed57da03"),
							mustDecodeString("3c4518eb4faa8ab01503057933932f33503e7693e05bb20948efd525be5850df"),
						},
						bc.Hash{V0: 13464118406972499748, V1: 5083224803004805715, V2: 16263625389659454272, V3: 9428032044180324575},
						*consensus.BTMAssetID, 99439999900, 2, mustDecodeString("0014a5f3f1941449c6072ade0ec5a66fad1417124f03"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 89439999900, mustDecodeString("0014ca1f877c2787f746a4473adac932171dd18d55d7"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 19900000000, mustDecodeString("00145ade29df622cc68d0473aa1a20fb89690451c66e"), nil),
				},
			},
			err: nil,
		},
		{
			desc: "multi utxo, single sign, non asset, btm stanard transaction, insufficient gas",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 595,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("4a8bf559f3c334ad23ed0aadab22dd3a4a8260488b1632dee16f75cac5c0ade674f2938776459414ab4d4e43622290507ff750a3fb563a25ee9a72386bfbe207"),
							mustDecodeString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c"), nil),
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("b4f6876a97c8e6bd7e038b476fb6fd07cdd6cfcf7d661dfab796b5e2c777b3de166495de4fba2aa154af844ed6a3d51c26742241edb0d5d107fc52dfff0f6305"),
							mustDecodeString("e5966eee4092eeefdd805b06f2ad368bb9392edec20998993ebe2a929052c1ce"),
						},
						bc.Hash{V0: 17091584763764411831, V1: 2315724244669489432, V2: 4322938623810388342, V3: 11167378497724951792},
						*consensus.BTMAssetID, 99960000000, 1, mustDecodeString("0014cfbccfac5018ad4b4bfbcb1fab834e3c85037460"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("00144b5637cc25b188136f440484f210541fa2a7ce64"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 89960000000, mustDecodeString("0014c7271a69dba57331b36221118dfeb1b1793933df"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 20000000000, mustDecodeString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049"), nil),
				},
			},
			err: vm.ErrRunLimitExceeded,
		},
		{
			desc: "single utxo, multi sign, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 396,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("6aa2dc1535e99a7b24eb215081613bbc9152cb073ebffc4846c819490f0836dec172b4893287cb026d230c0e462f237825fdcca29fde66cf096642a2844c860b"),
							mustDecodeString("28125a89064f60333d3bc213a18e13a4d35038bcaafc1ff7349e24607d6b98c755a33c74f44830f86d272da57dac77cca681672dfd12431004b7abaebeabad00"),
							mustDecodeString("ae20ed8e23a24df18f7a3dc148b9fa25cf849cdf233516528af683fe8a950ae31cff2015fb5ccf17b225be43161037d9b671cc26344c65e6fee1519ef66379c25f73905252ad"),
						},
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						*consensus.BTMAssetID, 89220000000, 1, mustDecodeString("00203adcfdf4a4e7e2b27d5b9f6eeafeebb4db229f73db244931c69feb54775d8510"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 79220000000, mustDecodeString("00206205ec178dc1ac6ea05ea01bb0fcda6aa978173026fa75204a101bdad7bd6b48"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, mustDecodeString("0014414eb62abda9a9191f9cba5d7e38d92f3e91e268"), nil),
				},
			},
			err: nil,
		},
		{
			desc: "single utxo, retire, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 309,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("3501a5dbfc05872d4da038893ecf3d2a776e3a805b3ae3f83bd461064a318ad7aa11d897e52b571a037347a8d4e3ccd0b7cdcda232af3142fa1329ac290f2509"),
							mustDecodeString("a0513ea06993680e7e639400ab051bbedeac675ed8f2085d4d6379fb8000bdb4"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014d3b239cfb5aa0b3302872b92682623ed408d0afc"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 11718900000, mustDecodeString("0014085a02ecdf934a56343aa59a3dec9d9feb86ee43"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 90000000, []byte{byte(vm.OP_FAIL)}, nil),
				},
			},
			err: nil,
		},
		{
			desc: "single utxo, single sign, issuance, spend, retire, btm stanard transaction, gas sufficient",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 601,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("cbd46e05c84c23a1ef7c27dcef189f1d386088a49df1c345de5d988bbe7d557d8406c596d6caea161473f5e3467d3d1883a75a559c71a0be5f2452b8aa3e510e"),
							mustDecodeString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014f233267911e94dc74df706fe3b697273e212d545"), []byte{}),
					types.NewIssuanceInput(
						mustDecodeString("fd0aec4229deb281"),
						10000000000,
						mustDecodeString("ae20ae12def13abe1295b477c24aa8ad5f1a60a60927c355e046bcd078a133b8d94c5151ad"),
						[][]byte{
							mustDecodeString("d6b284c40289a9417b53b3351483112ccf8a5f35cdb7812684a6df471a461774b118e546444dd16ddb469b536b96b6185430faff75ffae4661b9853657a2de0f"),
						},
						mustDecodeString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("00147d6b00edfbbc758a5da6130a5fa1a4cfec8422c3"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, []byte{byte(vm.OP_FAIL)}, nil),
					types.NewOriginalTxOutput(bc.AssetID{V0: 18275849036764733644, V1: 7408071477801754980, V2: 2368297496240756305, V3: 216480183129600045}, 10000000000, mustDecodeString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049"), nil),
				},
			},
			err: nil,
		},
		{
			desc: "single utxo, single sign, issuance, spend, retire, btm stanard transaction, gas insufficient",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 601,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("23ca3a6f8474b1b9ab8b77fcf3cf3fd9dfa761dff4e5d8551a72307dc065cd19100f3ca9fcca4df2f8842b71dba2fd29b73c1b06b3d8bddc2a71e8cc18842a04"),
							mustDecodeString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c"), nil),
					types.NewIssuanceInput(
						mustDecodeString("4b6afc9344c3ce63"),
						10000000000,
						mustDecodeString("ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad"),
						[][]byte{
							mustDecodeString("e8f301f7bd3b1e4ca85f1f8acda3a91fb73e717c096b8b82b2c7ed9d25170c0f9fcd9b5e8039094bd1174886f1b5428272eb6c2af03946bf3c2037a4b499c77107b94b96a92088a0d0d3b15559b3a253a4f5f9c7efba233ab0f6896bec23adc6a816c350e08f6b8ac5bc23eb5720173f9190805328af581f34a7fe561358d100"),
						},
						mustDecodeString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("001482b7991d64d001009b673ffe3ca2b35eab14f142"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 10000000000, []byte{byte(vm.OP_FAIL)}, nil),
					types.NewOriginalTxOutput(bc.AssetID{V0: 8879089148261671560, V1: 16875272676673176923, V2: 14627348561007036053, V3: 5774520766896450836}, 10000000000, mustDecodeString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049"), nil),
				},
			},
			err: vm.ErrRunLimitExceeded,
		},
		{
			desc: "btm stanard transaction check signature is not passed",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 331,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("298fbf48459480914e19a0fc20440b095bd7f38d9f01c56bfc904b4ed4967a7b73f1fc4919f23a7806eeb834a89f8ce696500f4528e8f7bf29c8ee1f38a91e02"),
							mustDecodeString("5a260070d967d894a9c4a6e16670c2881ed4c225e12d93b0707156e71fce5bfd"),
						},
						bc.Hash{V0: 3485387979411255237, V1: 15603105575416882039, V2: 5974145557334619041, V3: 16513948410238218452},
						*consensus.BTMAssetID, 21819700000, 0, mustDecodeString("001411ef7695d46e1f9288d996c3daa6ff4d956ac355"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 11818900000, mustDecodeString("001415c956112c2b46354690e36051803cc9d5a8f26b"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 10000000000, mustDecodeString("00149c9dd93184cc34ac5d47c145c5af3df852235aad"), nil),
				},
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "non btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 508,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{mustDecodeString("8fb54963e411982c0855924b22a07ea432df0f78a90558d7a759b93275991da18e101b8bcd2227b07ed5666732826e19d34e715dd432bfac49c5e7a5833b2e0a")},
						bc.Hash{V0: 13727785470566991667, V1: 17422390991613608658, V2: 10016033157382430074, V3: 8274310611876171875},
						bc.AssetID{V0: 986236576456443635, V1: 13806502593573493203, V2: 9657495453304566675, V3: 15226142438973879401},
						1000,
						1,
						mustDecodeString("207642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e7403ae7cac00c0"),
						nil),
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("ae8eb568b080d43d05a4f84f92e9f189937b490a34416ec4b111b995af45f9ac4d776af528294f22465c1695949746bc6ca6a605ee5cc7e9c394e316c3fca30c"),
							mustDecodeString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 5430419158397285610, V1: 15989125147582690097, V2: 3140150800656736345, V3: 4704385074037173738},
						*consensus.BTMAssetID, 9800000000, 2, mustDecodeString("0014f233267911e94dc74df706fe3b697273e212d545"),
						nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(
						bc.AssetID{V0: 986236576456443635, V1: 13806502593573493203, V2: 9657495453304566675, V3: 15226142438973879401},
						1000,
						mustDecodeString("001437e1aec83a4e6587ca9609e4e5aa728db7007449"),
						nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 9750000000, mustDecodeString("0014ec75fda5c727cb0d41137ab62afbf9070a405744"), nil),
				},
			},
			err: nil,
		},
	}

	for i, c := range cases {
		_, err := ValidateTx(types.MapTx(c.txData), mockBlock(), converter)
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s; validationState is:\n", i, c.desc, err, c.err)
		}
	}
}

func mustDecodeString(hexString string) []byte {
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	return bytes
}
