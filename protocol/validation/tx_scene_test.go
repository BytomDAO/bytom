package validation

import (
	"encoding/hex"
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
)

func TestValidateTx(t *testing.T) {
	cases := []struct {
		desc     string
		txData   *types.TxData
		gasValid bool
		err      error
	}{
		{
			desc: "single utxo, single sign, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 331,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("298fbf48459480914e19a0fc20440b095bd7f38d9f01c56bfc904b4ed4967a7b73f1fc4919f23a7806eeb834a89f8ce696500f4528e8f7bf29c8ee1f38a91e01"),
							mustDecodeString("5a260070d967d894a9c4a6e16670c2881ed4c225e12d93b0707156e71fce5bfd"),
						},
						bc.Hash{V0: 3485387979411255237, V1: 15603105575416882039, V2: 5974145557334619041, V3: 16513948410238218452},
						*consensus.BTMAssetID, 21819700000, 0, mustDecodeString("001411ef7695d46e1f9288d996c3daa6ff4d956ac355")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 11818900000, mustDecodeString("001415c956112c2b46354690e36051803cc9d5a8f26b")),
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, mustDecodeString("00149c9dd93184cc34ac5d47c145c5af3df852235aad")),
				},
			},
			gasValid: true,
			err:      nil,
		},
		{
			desc: "multi utxo, single sign, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 595,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("d488321eff213793fb685749a65b945b4d32f08774c27461e0dda265580e9d4582f4b210756b7f8a5b4a64bde531076e92244e12c145c9b54012134cebf9e402"),
							mustDecodeString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c")),
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("5d528bdb13b93c26245dc90c1fe51265555eb22a34fa013649db9aa874eb7770c6c4016320017224efdecf5fee39b682151f881f82c2c7195fe444ac5966140e"),
							mustDecodeString("563cb0eedf2a2891926dfaa0b9ac20913c67a066517f06b1f77c5ab527a8a8c4"),
						},
						bc.Hash{V0: 13464118406972499748, V1: 5083224803004805715, V2: 16263625389659454272, V3: 9428032044180324575},
						*consensus.BTMAssetID, 99439999900, 2, mustDecodeString("001419f79910f29df2ef80ec10d24c78e2009ed19302")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(*consensus.BTMAssetID, 89439999900, mustDecodeString("0014ca1f877c2787f746a4473adac932171dd18d55d7")),
					types.NewTxOutput(*consensus.BTMAssetID, 19900000000, mustDecodeString("00145ade29df622cc68d0473aa1a20fb89690451c66e")),
				},
			},
			gasValid: true,
			err:      nil,
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
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c")),
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("b4f6876a97c8e6bd7e038b476fb6fd07cdd6cfcf7d661dfab796b5e2c777b3de166495de4fba2aa154af844ed6a3d51c26742241edb0d5d107fc52dfff0f6305"),
							mustDecodeString("e5966eee4092eeefdd805b06f2ad368bb9392edec20998993ebe2a929052c1ce"),
						},
						bc.Hash{V0: 17091584763764411831, V1: 2315724244669489432, V2: 4322938623810388342, V3: 11167378497724951792},
						*consensus.BTMAssetID, 99960000000, 1, mustDecodeString("0014cfbccfac5018ad4b4bfbcb1fab834e3c85037460")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("00144b5637cc25b188136f440484f210541fa2a7ce64")),
					types.NewTxOutput(*consensus.BTMAssetID, 89960000000, mustDecodeString("0014c7271a69dba57331b36221118dfeb1b1793933df")),
					types.NewTxOutput(*consensus.BTMAssetID, 20000000000, mustDecodeString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049")),
				},
			},
			gasValid: false,
			err:      vm.ErrRunLimitExceeded,
		},
		{
			desc: "single utxo, multi sign, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 396,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("abc55905b5c477f424ea5ce88bbd00376f18f525850b7b74f54e94e7999edbe5ded9e9f5d8f1319470e9a38540bbaa6bbe67aacc8227c898ae30b9ac15f8dc0b"),
							mustDecodeString("ae203f56f71972918585ece56a21f77c3e9101ce14c75038b65454e10960266cceba20c9927f289b57c647578d07904a9d34597079d80e300df023a26658a770f611545152ad"),
						},
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						*consensus.BTMAssetID, 89220000000, 1, mustDecodeString("0020ff726649e34c921ff61a97090fc62054f339597acfc710197bb0133e18a19c5c")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 79220000000, mustDecodeString("00206205ec178dc1ac6ea05ea01bb0fcda6aa978173026fa75204a101bdad7bd6b48")),
					types.NewTxOutput(*consensus.BTMAssetID, 9900000000, mustDecodeString("0014414eb62abda9a9191f9cba5d7e38d92f3e91e268")),
				},
			},
			gasValid: true,
			err:      nil,
		},
		{
			desc: "single utxo, retire, non asset, btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 309,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("f0009a0fa67238f6dfbb208282f509fb460531f43f74809e0226af2ff064607fad8a2506779e717a5f7848bbc3abdfa724148a9df46426027f201a4dfec27809"),
							mustDecodeString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 11718900000, mustDecodeString("0014085a02ecdf934a56343aa59a3dec9d9feb86ee43")),
					types.NewTxOutput(*consensus.BTMAssetID, 90000000, []byte{byte(vm.OP_FAIL)}),
				},
			},
			gasValid: true,
			err:      nil,
		},
		{
			desc: "single utxo, single sign, issuance, spend, retire, btm stanard transaction, gas sufficient",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 601,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("8aab6052cb935384ac8fcbd4c0857cbce2e19825a002635d0b242757f17e5fdd148d83eb3837baf91754bf539cd08e29f66975f4bc9843ac00e280f228026105"),
							mustDecodeString("ca85ea98011ddd592d1f081ebd2a91ac0f4238784222ed85b9d95aeb654f1cf1"),
						},
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c")),
					types.NewIssuanceInput(
						mustDecodeString("fd0aec4229deb281"),
						10000000000,
						mustDecodeString("ae2054a71277cc162eb3eb21b5bd9fe54402829a53b294deaed91692a2cd8a081f9c5151ad"),
						[][]byte{
							mustDecodeString("e8f301f7bd3b1e4ca853b15559b3a253a4f5f9c7efba233ab0f6896bec23adc6a816c350e08f6b8ac5bc23eb5720173f9190805328af581f34a7fe561358d100"),
						},
						mustDecodeString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("00147d6b00edfbbc758a5da6130a5fa1a4cfec8422c3")),
					types.NewTxOutput(*consensus.BTMAssetID, 9900000000, []byte{byte(vm.OP_FAIL)}),
					types.NewTxOutput(bc.AssetID{V0: 8879089148261671560, V1: 16875272676673176923, V2: 14627348561007036053, V3: 5774520766896450836}, 10000000000, mustDecodeString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049")),
				},
			},
			gasValid: true,
			err:      nil,
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
						*consensus.BTMAssetID, 11818900000, 0, mustDecodeString("0014e6e1f8b11f1cfb7609037003b90f64837afd272c")),
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
					types.NewTxOutput(*consensus.BTMAssetID, 1818900000, mustDecodeString("001482b7991d64d001009b673ffe3ca2b35eab14f142")),
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, []byte{byte(vm.OP_FAIL)}),
					types.NewTxOutput(bc.AssetID{V0: 8879089148261671560, V1: 16875272676673176923, V2: 14627348561007036053, V3: 5774520766896450836}, 10000000000, mustDecodeString("0014447e597c1c326ad1a639f8023d3f87ae22a4e049")),
				},
			},
			gasValid: false,
			err:      vm.ErrRunLimitExceeded,
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
						*consensus.BTMAssetID, 21819700000, 0, mustDecodeString("001411ef7695d46e1f9288d996c3daa6ff4d956ac355")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 11818900000, mustDecodeString("001415c956112c2b46354690e36051803cc9d5a8f26b")),
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, mustDecodeString("00149c9dd93184cc34ac5d47c145c5af3df852235aad")),
				},
			},
			gasValid: false,
			err:      vm.ErrFalseVMResult,
		},
		{
			desc: "non btm stanard transaction",
			txData: &types.TxData{
				Version:        1,
				SerializedSize: 508,
				Inputs: []*types.TxInput{
					types.NewSpendInput(
						[][]byte{mustDecodeString("585f298f2396c3b1be616b6eb48b21c7ec2b25fa4daf7256e970e0a55658c04cbcb406ed41e6b184732187daf0627ea805b24098785c80979edf4d4fc2b8100c")},
						bc.Hash{V0: 13727785470566991667, V1: 17422390991613608658, V2: 10016033157382430074, V3: 8274310611876171875},
						bc.AssetID{V0: 986236576456443635, V1: 13806502593573493203, V2: 9657495453304566675, V3: 15226142438973879401},
						1000,
						1,
						mustDecodeString("206dbca07ff0a6025612c835423daadd4460c3a2ed9a65622ba8025dfd3388238c7403ae7cac00c0")),
					types.NewSpendInput(
						[][]byte{
							mustDecodeString("4ef8f5a377c166b9fb4efa221894f06194b6b7bc277e613ad75b442929a417bb278ee347586e8f06b20c9b759263c981f03d00253f49753fde88dc8b39ccb10e"),
							mustDecodeString("1381d35e235813ad1e62f9a602c82abee90565639cc4573568206b55bcd2aed9"),
						},
						bc.Hash{V0: 5430419158397285610, V1: 15989125147582690097, V2: 3140150800656736345, V3: 4704385074037173738},
						*consensus.BTMAssetID, 9800000000, 2, mustDecodeString("0014cb9f2391bafe2bc1159b2c4c8a0f17ba1b4dd94e")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(
						bc.AssetID{V0: 986236576456443635, V1: 13806502593573493203, V2: 9657495453304566675, V3: 15226142438973879401},
						1000,
						mustDecodeString("001437e1aec83a4e6587ca9609e4e5aa728db7007449")),
					types.NewTxOutput(*consensus.BTMAssetID, 9750000000, mustDecodeString("0014ec75fda5c727cb0d41137ab62afbf9070a405744")),
				},
			},
			gasValid: true,
			err:      nil,
		},
	}

	for i, c := range cases {
		gasStatus, err := ValidateTx(types.MapTx(c.txData), mockBlock())
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s; validationState is:\n", i, c.desc, err, c.err)
		}
		if c.gasValid != gasStatus.GasValid {
			t.Errorf("#%d got GasValid %t, want %t", i, gasStatus.GasValid, c.gasValid)
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
