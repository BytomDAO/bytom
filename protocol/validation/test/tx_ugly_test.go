package test

import (
	"encoding/hex"
	"math"
	"testing"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
)

func TestValidateUglyTx(t *testing.T) {
	singleSignInst := &signingInst{
		rootPrvKeys: []string{
			"38d2c44314c401b3ea7c23c54e12c36a527aee46a7f26b82443a46bf40583e439dea25de09b0018b35a741d8cd9f6ec06bc11db49762617485f5309ab72a12d4",
		},
		quorum:           1,
		keyIndex:         1,
		ctrlProgramIndex: 1,
		change:           false,
	}
	multiSignInst := &signingInst{
		rootPrvKeys: []string{
			"a080aca2d9d7948d005c92d0729c618e56fb5551a52dfa04dc4caaf3c8b8a94c89a9795f5bbfd2b885ce7a9d3e3efa5386436c3681b21f9263a0b0a544346b48",
			"105295324626e33bb7d8e8a57c6a0aa495346d7fc342a4891ece00424494cf48f75cefa0f8c61674a12238cfa711b4bc26cb22f38b6e2206c691b83943a58312",
			"c02bb73d1aee56f8935fb7704f71f668eb37ec223baf5723b38a186669b465427d1bdbc2c4397c1259d12b6229aaf6154aaccdeb8addce3a780a1cbc1025ad25",
			"a0c2225685e4c4439f12c264d1573db063ddbc929d4b8a3e1641e8abb4df504a56b1200b9925138d79febe6e1156fcfaf0d1878f25cbccc5db4c8fea55bde198",
			"20d06d4fd261ab554e01104f019392f89566acace727e6bb6de4544aa3a6b248480232155332e6e5de10a62e4a9a4c1d9e3b7f9cb4fd196142ef1d080b8bbaec",
		},
		quorum:           3,
		keyIndex:         1,
		ctrlProgramIndex: 1,
		change:           false,
	}
	cases := []struct {
		category string
		desc     string
		insts    []*signingInst
		txData   types.TxData
		gasValid bool
		err      bool
	}{
		{
			category: "fee insufficient",
			desc:     "sum of btm output greater than btm input",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 10000000001, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "fee insufficient",
			desc:     "sum of btm output equals to input btm",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "fee insufficient",
			desc:     "sum of btm input greater than btm output, but still insufficient",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000001, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "fee insufficient",
			desc:     "no btm input",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
		{
			category: "input output unbalance",
			desc:     "only has btm input, no output",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000001, 0, nil),
				},
				Outputs: []*types.TxOutput{},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "issuance asset, no corresponding output",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "issuance asset A, output asset B",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707e"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of output asset A greater than issuance asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 20000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of output asset A less than issuance asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 5000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of retire asset A greater than issuance asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 20000000000, testutil.MustDecodeHexString("6a")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of retire asset A less than issuance asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 5000000000, testutil.MustDecodeHexString("6a")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "spend asset A, no corresponding output",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "spend asset A, output asset B",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707e"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of output asset A greater than spend asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 20000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of output asset A less than spend asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 5000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of retire asset A greater than spend asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 20000000000, testutil.MustDecodeHexString("6a")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "sum of retire asset A less than spend asset A",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 5000000000, testutil.MustDecodeHexString("6a")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "use retired utxo",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, testutil.MustDecodeHexString("6a")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "input utxo is zero",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 0, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 0, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "input output unbalance",
			desc:     "no btm input",
			txData: types.TxData{
				Version: 1,
				Inputs:  []*types.TxInput{},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 10, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "spend btm input overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, math.MaxUint64, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						*consensus.BTMAssetID, 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "spend non btm input overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), math.MaxInt64, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 100, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						*consensus.BTMAssetID, 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 100, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "issuance non btm input overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						*consensus.BTMAssetID, 10000000000, 1, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						math.MaxInt64,
						nil,
						[][]byte{
							testutil.MustDecodeHexString("e8f301f7bd3b1e4ca853b15559b3a253a4f5f9c7efba233ab0f6896bec23adc6a816c350e08f6b8ac5bc23eb5720173f9190805328af581f34a7fe561358d100"),
						},
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "sum of spend and issuance non btm input overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						*consensus.BTMAssetID, 10000000000, 1, nil),
					types.NewIssuanceInput( // assetID: 97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						math.MaxInt64,
						nil,
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 100, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 100, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "spend btm output overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, math.MaxUint64, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "retire btm output overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, math.MaxUint64, testutil.MustDecodeHexString("6a")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "non btm output overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(*consensus.BTMAssetID, math.MaxUint64, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "retire non btm output overflow",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(*consensus.BTMAssetID, math.MaxUint64, testutil.MustDecodeHexString("6a")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "overflow",
			desc:     "output with over range amount but sum in equal",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 100000000, 0, nil),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 18446744073609551616, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(*consensus.BTMAssetID, 18446744073609551616, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(*consensus.BTMAssetID, 290000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "verify signature fail",
			desc:     "btm single sign",
			insts:    []*signingInst{singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, testutil.MustDecodeHexString("00140876db6ca8f4542a836f0edd42b87d095d081182")), // wrong control program
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "verify signature fail",
			desc:     "btm multi sign",
			insts:    []*signingInst{multiSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, testutil.MustDecodeHexString("00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66")), // wrong control program
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: false,
		},
		{
			category: "verify signature fail",
			desc:     "spend non btm single sign",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, testutil.MustDecodeHexString("00140876db6ca8f4542a836f0edd42b87d095d081182")), // wrong control program
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
		{
			category: "verify signature fail",
			desc:     "spend non btm multi sign",
			insts:    []*signingInst{singleSignInst, multiSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(nil,
						bc.Hash{V0: 6970879411704044573, V1: 10086395903308657573, V2: 10107608596190358115, V3: 8645856247221333302},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 1, testutil.MustDecodeHexString("00140876db6ca8f4542a836f0edd42b87d095d081182")), // wrong control program
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
		{
			category: "verify signature fail",
			desc:     "issuance non btm single sign",
			insts:    []*signingInst{singleSignInst, multiSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput(
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						// wrong issuance program
						testutil.MustDecodeHexString("ae20c38173d800e62f63bd08cfaa9bc905e4a34a61ad841d7ad6c70ead0fb48196995151ad"),
						nil,
						nil,
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("bf5f8da2334590ee095148ccdcf4d806b26a47a6d9e9e857ef6c2de79aee4f14"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
		{
			category: "verify signature fail",
			desc:     "issuance non btm multi sign",
			insts:    []*signingInst{singleSignInst, multiSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewIssuanceInput(
						testutil.MustDecodeHexString("fd0aec4229deb281"),
						10000000000,
						// wrong issuance program
						testutil.MustDecodeHexString("ae20ce8639c5dc70cb2b12f89a057670602eb013fc54a10ce22bd4691c62cf546b7b2081bdd879bcbce7f58e1731841c6b3deac242efa00e75124fe559fa531c0c5bb820b40b6eec74288ee4bae67191f135512454b52640cfd7be95dc84be0f02281dce20247b6e6f9230a987ef61c66820268e7b766d28c1ce7aa2c550b34e294167f340205096211460415888768a48b121013711aa711634bb9ff7341a7bd072c31525875355ad"),
						nil,
						testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
					),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("776f0a421e9176a03061d388aff4ab3b1bcd32e53a090d593a466706c69e3d3f"), 10000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
		{
			category: "double spend",
			desc:     "btm asset double spend",
			insts:    []*signingInst{singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, testutil.MustDecodeHexString("001420a1af4fc11399e6cd7253abf1bbd4d0af17daad")),
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, testutil.MustDecodeHexString("001420a1af4fc11399e6cd7253abf1bbd4d0af17daad")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 19000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
		{
			category: "double spend",
			desc:     "non btm asset double spend",
			insts:    []*signingInst{singleSignInst, singleSignInst, singleSignInst},
			txData: types.TxData{
				Version: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput(nil,
						bc.Hash{V0: 14760873410800997144, V1: 1698395500822741684, V2: 5965908492734661392, V3: 9445539829830863994},
						*consensus.BTMAssetID, 10000000000, 0, nil),
					types.NewSpendInput(
						nil,
						bc.Hash{V0: 3485387979411255237, V1: 15603105575416882039, V2: 5974145557334619041, V3: 16513948410238218452},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 0, testutil.MustDecodeHexString("001420a1af4fc11399e6cd7253abf1bbd4d0af17daad")),
					types.NewSpendInput(
						nil,
						bc.Hash{V0: 3485387979411255237, V1: 15603105575416882039, V2: 5974145557334619041, V3: 16513948410238218452},
						testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 10000000000, 0, testutil.MustDecodeHexString("001420a1af4fc11399e6cd7253abf1bbd4d0af17daad")),
				},
				Outputs: []*types.TxOutput{
					types.NewTxOutput(*consensus.BTMAssetID, 9000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
					types.NewTxOutput(testutil.MustDecodeAsset("97575084e5161406a0977da729fbf51ad230e0ff0aec607a97e4336611c8707f"), 20000000000, testutil.MustDecodeHexString("00145931e1b7b65897f47845ac08fc136e0c0a4ff166")),
				},
			},
			err:      true,
			gasValid: true,
		},
	}

	for i, c := range cases {
		mockCtrlProgram(c.txData, c.insts)

		c.txData.SerializedSize = 1

		tx := types.NewTx(c.txData)
		mockSignTx(tx, c.insts)
		bcTx := types.MapTx(&c.txData)

		gasStatus, err := validation.ValidateTx(bcTx, &bc.Block{
			BlockHeader:  &bc.BlockHeader{Height: 1},
			Transactions: []*bc.Tx{bcTx},
		})
		if !c.err && err != nil {
			t.Errorf("case #%d (%s) expect no error, got error %s", i, c.desc, err)
		}

		if c.err && err == nil {
			t.Errorf("case #%d (%s) expect error, got no error", i, c.desc)
		}

		if c.gasValid != gasStatus.GasValid {
			t.Errorf("case #%d (%s) got GasValid %t, want %t", i, c.desc, gasStatus.GasValid, c.gasValid)
		}
	}
}

type signingInst struct {
	rootPrvKeys      []string
	quorum           int
	keyIndex         uint64
	ctrlProgramIndex uint64
	change           bool
}

func mockCtrlProgram(txData types.TxData, insts []*signingInst) {
	for i, input := range txData.Inputs {
		_, xPubs := mustGetRootKeys(insts[i].rootPrvKeys)

		switch inp := input.TypedInput.(type) {
		case *types.SpendInput:
			if inp.ControlProgram != nil {
				continue
			}
			acc := &account.Account{Signer: &signers.Signer{KeyIndex: insts[i].keyIndex, DeriveRule: signers.BIP0044, XPubs: xPubs, Quorum: insts[i].quorum}}
			program, err := account.CreateCtrlProgram(acc, insts[i].ctrlProgramIndex, insts[i].change)
			if err != nil {
				panic(err)
			}
			inp.ControlProgram = program.ControlProgram
		case *types.IssuanceInput:
			if inp.IssuanceProgram != nil {
				continue
			}
			assetSigner, err := signers.Create("asset", xPubs, insts[i].quorum, insts[i].keyIndex, signers.BIP0032)
			if err != nil {
				panic(err)
			}

			path := signers.GetBip0032Path(assetSigner, signers.AssetKeySpace)
			derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
			derivedPKs := chainkd.XPubKeys(derivedXPubs)

			issuanceProg, err := vmutil.P2SPMultiSigProgramWithHeight(derivedPKs, insts[i].quorum, 0)
			if err != nil {
				panic(err)
			}

			inp.IssuanceProgram = issuanceProg
		}
	}
}

func mockSignTx(tx *types.Tx, insts []*signingInst) {
	for i, input := range tx.TxData.Inputs {
		if input.Arguments() != nil {
			continue
		}
		var arguments [][]byte
		inst := insts[i]
		switch inp := input.TypedInput.(type) {
		case *types.SpendInput:
			path, err := signers.Path(&signers.Signer{KeyIndex: inst.keyIndex, DeriveRule: signers.BIP0044}, signers.AccountKeySpace, inst.change, inst.ctrlProgramIndex)
			if err != nil {
				panic(err)
			}

			xPrvs, xPubs := mustGetRootKeys(inst.rootPrvKeys)
			for _, xPrv := range xPrvs {
				childPrv := xPrv.Derive(path)
				sigHashBytes := tx.SigHash(uint32(i)).Byte32()
				arguments = append(arguments, childPrv.Sign(sigHashBytes[:]))
			}

			if len(xPrvs) == 1 {
				childPrv := xPrvs[0].Derive(path)
				derivePK := childPrv.XPub()
				arguments = append(arguments, derivePK.PublicKey())
			} else {
				derivedXPubs := chainkd.DeriveXPubs(xPubs, path)
				derivedPKs := chainkd.XPubKeys(derivedXPubs)
				script, err := vmutil.P2SPMultiSigProgram(derivedPKs, inst.quorum)
				if err != nil {
					panic(err)
				}

				arguments = append(arguments, script)
			}
			inp.Arguments = arguments
		case *types.IssuanceInput:
			path := signers.GetBip0032Path(&signers.Signer{KeyIndex: inst.keyIndex, DeriveRule: signers.BIP0032}, signers.AssetKeySpace)
			xPrvs, _ := mustGetRootKeys(inst.rootPrvKeys)
			for _, xPrv := range xPrvs {
				childPrv := xPrv.Derive(path)
				sigHashBytes := tx.SigHash(uint32(i)).Byte32()
				arguments = append(arguments, childPrv.Sign(sigHashBytes[:]))
			}
			inp.Arguments = arguments
		}
	}
}

func mustGetRootKeys(prvs []string) ([]chainkd.XPrv, []chainkd.XPub) {
	xPubs := make([]chainkd.XPub, len(prvs))
	xPrvs := make([]chainkd.XPrv, len(prvs))
	for i, xPrv := range prvs {
		xPrvBytes, err := hex.DecodeString(xPrv)
		if err != nil {
			panic(err)
		}

		if len(xPrvBytes) != 64 {
			panic("the size of xPrv must 64")
		}

		var dest [64]byte
		copy(dest[:], xPrv)
		xPrvs[i] = chainkd.XPrv(dest)
		xPubs[i] = xPrvs[i].XPub()
	}
	return xPrvs, xPubs
}
