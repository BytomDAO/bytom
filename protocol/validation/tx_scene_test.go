package validation

import (
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/testutil"
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
							testutil.MustDecodeHexString("d5380072fcf74cd9625c0ba77e9c6ae4604521889e3dd42fcba770f0e7523b2b2e2c0fe2218f75def7d7375bbc24e8b6c51afc0d7900acac3849b34245b46703"),
							testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 2},
						*consensus.BTMAssetID, 1000000000, 0, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), [][]byte{}),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 100, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), [][]byte{}),
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
							testutil.MustDecodeHexString("725dd970e2654fc75ef9affbd31ca20f973c28a7101d5f35c064ee6a1afe91fb75495ab4e6a83d1b213c4eed87d987bc3d4256782f33b62d418abaeaae7f0a0d"),
							testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), nil),
					types.NewSpendInput(
						[][]byte{
							testutil.MustDecodeHexString("c714d495298d02cdaa285a740eedca6f05df1dfbc8eb5d17498b3e8b8feacd51f9bac6c302dbe8157b3a107ea41c742dae42d1b5a60f46804698eb7cf578d50e"),
							testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 13464118406972499748, V1: 5083224803004805715, V2: 16263625389659454272, V3: 9428032044180324575},
						*consensus.BTMAssetID, 99439999900, 2, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 89439999900, testutil.MustDecodeHexString("0014ca1f877c2787f746a4473adac932171dd18d55d7"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 19900000000, testutil.MustDecodeHexString("00145ade29df622cc68d0473aa1a20fb89690451c66e"), nil),
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
							testutil.MustDecodeHexString("4a8bf559f3c334ad23ed0aadab22dd3a4a8260488b1632dee16f75cac5c0ade674f2938776459414ab4d4e43622290507ff750a3fb563a25ee9a72386bfbe207"),
							testutil.MustDecodeHexString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, testutil.MustDecodeHexString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c"), nil),
					types.NewSpendInput(
						[][]byte{
							testutil.MustDecodeHexString("b4f6876a97c8e6bd7e038b476fb6fd07cdd6cfcf7d661dfab796b5e2c777b3de166495de4fba2aa154af844ed6a3d51c26742241edb0d5d107fc52dfff0f6305"),
							testutil.MustDecodeHexString("e5966eee4092eeefdd805b06f2ad368bb9392edec20998993ebe2a929052c1ce"),
						},
						bc.Hash{V0: 17091584763764411831, V1: 2315724244669489432, V2: 4322938623810388342, V3: 11167378497724951792},
						*consensus.BTMAssetID, 99960000000, 1, testutil.MustDecodeHexString("0014cfbccfac5018ad4b4bfbcb1fab834e3c85037460"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, testutil.MustDecodeHexString("00144b5637cc25b188136f440484f210541fa2a7ce64"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 89960000000, testutil.MustDecodeHexString("0014c7271a69dba57331b36221118dfeb1b1793933df"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 20000000000, testutil.MustDecodeHexString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049"), nil),
				},
			},
			err: vm.ErrRunLimitExceeded,
		},
		{
			desc: "single utxo, retire, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 309,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							testutil.MustDecodeHexString("62313a8ae7bc039bf02df721bcf6d0581e16d0b23e097f96d3a107c22c6d75fc1e5ec41ceaa4104e38c97204cfc742f49fea95cbd06a9a5a19ea26d0c334c701"),
							testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 11718900000, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), nil),
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

							testutil.MustDecodeHexString("86416e48fe371d4f07afea76aef204d9acb2c1fff6742499d50feda314bc49f478cd28e9d6a4cdba6592da5e5b819cf6c3d40ad6326192ef3e2fcc6f6bfd4509"),
							testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"), [][]byte{}),
					types.NewIssuanceInput(
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						testutil.MustDecodeHexString("51"),
						[][]byte{},
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, testutil.MustDecodeHexString("00147d6b00edfbbc758a5da6130a5fa1a4cfec8422c3"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, []byte{byte(vm.OP_FAIL)}, nil),
					types.NewOriginalTxOutput(testutil.MustDecodeAsset("2e1e4db789f0a23ccf98038b01ba2949634c940a3f01154b5f73ca7f3ebef7c1"), 10000000000, testutil.MustDecodeHexString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049"), nil),
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
							testutil.MustDecodeHexString("23ca3a6f8474b1b9ab8b77fcf3cf3fd9dfa761dff4e5d8551a72307dc065cd19100f3ca9fcca4df2f8842b71dba2fd29b73c1b06b3d8bddc2a71e8cc18842a04"),
							testutil.MustDecodeHexString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, testutil.MustDecodeHexString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c"), nil),
					types.NewIssuanceInput(
						testutil.MustDecodeHexString("4b6afc9344c3ce63"),
						10000000000,
						testutil.MustDecodeHexString("ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad"),
						[][]byte{
							testutil.MustDecodeHexString("e8f301f7bd3b1e4ca85f1f8acda3a91fb73e717c096b8b82b2c7ed9d25170c0f9fcd9b5e8039094bd1174886f1b5428272eb6c2af03946bf3c2037a4b499c77107b94b96a92088a0d0d3b15559b3a253a4f5f9c7efba233ab0f6896bec23adc6a816c350e08f6b8ac5bc23eb5720173f9190805328af581f34a7fe561358d100"),
						},
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 1818900000, testutil.MustDecodeHexString("001482b7991d64d001009b673ffe3ca2b35eab14f142"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 10000000000, []byte{byte(vm.OP_FAIL)}, nil),
					types.NewOriginalTxOutput(bc.AssetID{V0: 8879089148261671560, V1: 16875272676673176923, V2: 14627348561007036053, V3: 5774520766896450836}, 10000000000, testutil.MustDecodeHexString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049"), nil),
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
							testutil.MustDecodeHexString("298fbf48459480914e19a0fc20440b095bd7f38d9f01c56bfc904b4ed4967a7b73f1fc4919f23a7806eeb834a89f8ce696500f4528e8f7bf29c8ee1f38a91e02"),
							testutil.MustDecodeHexString("5a260070d967d894a9c4a6e16670c2881ed4c225e12d93b0707156e71fce5bfd"),
						},
						bc.Hash{V0: 3485387979411255237, V1: 15603105575416882039, V2: 5974145557334619041, V3: 16513948410238218452},
						*consensus.BTMAssetID, 21819700000, 0, testutil.MustDecodeHexString("001411ef7695d46e1f9288d996c3daa6ff4d956ac355"), nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 11818900000, testutil.MustDecodeHexString("001415c956112c2b46354690e36051803cc9d5a8f26b"), nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 10000000000, testutil.MustDecodeHexString("00149c9dd93184cc34ac5d47c145c5af3df852235aad"), nil),
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
						[][]byte{testutil.MustDecodeHexString("f0b2326e8bb5ef8d069587a144edf7d249d2bb647ea7cbaed4a2ebb19865f6a9b8e63e911338a0b6cfb2992af25b3da4907682bc985305d737f91e96b9ac7b0b")},
						bc.Hash{V0: 13727785470566991667, V1: 17422390991613608658, V2: 10016033157382430074, V3: 8274310611876171875},
						bc.AssetID{V0: 986236576456443635, V1: 13806502593573493203, V2: 9657495453304566675, V3: 15226142438973879401},
						1000,
						1,
						testutil.MustDecodeHexString("207642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e7403ae7cac00c0"),
						nil),
					types.NewSpendInput(
						[][]byte{
							testutil.MustDecodeHexString("bc91faef22f5926c4042545e0b15649fe56c7e0a8d49a68b2460caf41511f1fb39fa82bc3d85f68658486d5b3ebf85351536898ce6e5f626a3bb111cad78dc02"),
							testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e"),
						},
						bc.Hash{V0: 5430419158397285610, V1: 15989125147582690097, V2: 3140150800656736345, V3: 4704385074037173738},
						*consensus.BTMAssetID, 9800000000, 2, testutil.MustDecodeHexString("0014f233267911e94dc74df706fe3b697273e212d545"),
						nil),
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(
						bc.AssetID{V0: 986236576456443635, V1: 13806502593573493203, V2: 9657495453304566675, V3: 15226142438973879401},
						1000,
						testutil.MustDecodeHexString("001437e1aec83a4e6587ca9609e4e5aa728db7007449"),
						nil),
					types.NewOriginalTxOutput(*consensus.BTMAssetID, 9750000000, testutil.MustDecodeHexString("0014ec75fda5c727cb0d41137ab62afbf9070a405744"), nil),
				},
			},
			err: nil,
		},
	}

	for i, c := range cases {
		if _, err := ValidateTx(types.MapTx(c.txData), mockBlock(), converter); rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s; validationState is:\n", i, c.desc, err, c.err)
		}
	}
}
