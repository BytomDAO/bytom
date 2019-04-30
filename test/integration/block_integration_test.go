package integration

import (
	"testing"

	"time"

	"github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/database"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/testutil"
)

var blockMap map[int][]*attachBlock

type attachBlock struct {
	block        *types.Block
	verifyResult []*bc.TxVerifyResult
}

func init() {
	consensus.ActiveNetParams = consensus.SoloNetParams

	blockMap = map[int][]*attachBlock{
		0: {
			{
				block:        config.GenesisBlock(),
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}},
			},
		},
		1: {
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            1,
						Version:           1,
						Timestamp:         1556431597,
						Nonce:             5,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("ce4fe9431cd0225b3a811f8f8ec922f2b07a921bb12a8dddae9a85540072c770"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0031")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}},
			},
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            1,
						Version:           1,
						Timestamp:         1556431697,
						Nonce:             36,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("ce4fe9431cd0225b3a811f8f8ec922f2b07a921bb12a8dddae9a85540072c770"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0031")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("00143d05e891b165b165afefa2e861e83a9745f80d8c")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}},
			},
		},
		2: {
			// only has coinbase transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431604,
						Nonce:             0,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}},
			},
			// with spend btm transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431604,
						Nonce:             12,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
						types.NewTx(types.TxData{
							Version:   1,
							TimeRange: 0,
							Inputs: []*types.TxInput{
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("7b4082c9d745c3f07dd07afb1f987960d2ef3ea2486741c3f3184751485f77d046df6670eba21020fcf9c7987c0c938384320dc21b0e116c62ae2597cb1fe109"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"),
									*consensus.BTMAssetID, 10000000000, 0,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("00143d05e891b165b165afefa2e861e83a9745f80d8c")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}},
			},
			// with btm retire transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						Nonce:             4,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
						types.NewTx(types.TxData{
							Version:   1,
							TimeRange: 0,
							Inputs: []*types.TxInput{
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("302035b362d80419cfed12cfc7d33d2ff7638c589ee2cf6573eb14b4d8cb4a63d7d1302589666dd6d1cd08185dbb2842407f3980bc2564705eda15680c984c05"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"),
									*consensus.BTMAssetID, 10000000000, 0,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("6a")), // retire
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}},
			},
			// with issuance transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						Nonce:             17,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
						types.NewTx(types.TxData{
							Version:   1,
							TimeRange: 0,
							Inputs: []*types.TxInput{
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("46cbb829b6a5bb9fc436c8e51bcbd9f0b3ed99ce97b2e0fac28879b4202c5a9eccaae39a4d18584f789a9427af3a2f09ff0360fb187e46ef172146a9b957ef0c"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"),
									*consensus.BTMAssetID, 10000000000, 0,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
								types.NewIssuanceInput(
									testutil.MustDecodeHexString("fd0aec4229deb281"),
									10000000000,
									testutil.MustDecodeHexString("ae20f25e8b73ffbc3a42300a43279fdf612d79e1936a6c614fc05a5adec9bba42dcd5151ad"),
									[][]byte{testutil.MustDecodeHexString("df9fabf4636904e017eefb7cdf2b4f08e29efbd4cfc41fe5b01a453191f0913489b19ad74272145824e92bd4843e91140cc5d1a6256f84981d1437ed4566a60b")},
									testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
								),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
								types.NewTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}},
			},
			// with issuance transaction but status fail is true
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						Nonce:             4,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
						types.NewTx(types.TxData{
							Version:   1,
							TimeRange: 0,
							Inputs: []*types.TxInput{
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("46cbb829b6a5bb9fc436c8e51bcbd9f0b3ed99ce97b2e0fac28879b4202c5a9eccaae39a4d18584f789a9427af3a2f09ff0360fb187e46ef172146a9b957ef0c"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"),
									*consensus.BTMAssetID, 10000000000, 0,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
								types.NewIssuanceInput(
									testutil.MustDecodeHexString("fd0aec4229deb281"),
									10000000000,
									testutil.MustDecodeHexString("ae20f25e8b73ffbc3a42300a43279fdf612d79e1936a6c614fc05a5adec9bba42dcd5151ad"),
									// invalid signature
									[][]byte{testutil.MustDecodeHexString("df9fabf4636904e017eefb7cdf2b4f08e29efbd4cfc41fe5b01a453191f0913489b19ad74272145824e92bd4843e91140cc5d1a6256f84981d1437ed4566a60c")},
									testutil.MustDecodeHexString("7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022222c0a20202273796d626f6c223a2022220a7d"),
								),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
								types.NewTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: true}},
			},
			// with non btm transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						Nonce:             4,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
						types.NewTx(types.TxData{
							Version:   1,
							TimeRange: 0,
							Inputs: []*types.TxInput{
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("afc4e24f0e0383e3fd78af3de189be3913faddbbd8cac8a8c9316bf9eb0866e83df3618cf4c7b4d091a79968a16377d422cbd8011f1f5e75ba201e173b68ad02"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"),
									*consensus.BTMAssetID, 10000000000, 0,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("cd6fb451102db667341438f20dbeabd30b343ed08d89625a8e27e82478e89ddea9e7d51f8a4036e0cc2602ac5fae0bdbfda025a0e2c12e3ddc8100b62461670b"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f22"),
									testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, 1,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
								types.NewTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}},
			},
			// with non btm transaction but status fail is true
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						Nonce:             12,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("2eaf7f40b0a0d4a5025f3d5d9b8589d3db1634f7b55089ca59253a9c587266b2"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0032")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
						types.NewTx(types.TxData{
							Version:   1,
							TimeRange: 0,
							Inputs: []*types.TxInput{
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("afc4e24f0e0383e3fd78af3de189be3913faddbbd8cac8a8c9316bf9eb0866e83df3618cf4c7b4d091a79968a16377d422cbd8011f1f5e75ba201e173b68ad02"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"),
									*consensus.BTMAssetID, 10000000000, 0,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
								types.NewSpendInput(
									// invalid signature
									[][]byte{
										testutil.MustDecodeHexString("cd6fb451102db667341438f20dbeabd30b343ed08d89625a8e27e82478e89ddea9e7d51f8a4036e0cc2602ac5fae0bdbfda025a0e2c12e3ddc8100b62461670c"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f22"),
									testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, 1,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
								),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
								types.NewTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: true}},
			},
		},
		3: {
			// the previous block is blockMap[2][0]
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            3,
						Version:           1,
						Timestamp:         1556431640,
						Nonce:             0,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("09c6064f4f1e7325440c45df03e97f97dbfbb66033315a384308256038af6c30"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0033")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}},
			},
			// the previous block is blockMap[2][2]
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            3,
						Version:           1,
						Timestamp:         1556431640,
						Nonce:             5,
						Bits:              2305843009214532812,
						PreviousBlockHash: testutil.MustDecodeHash("33f56264283cc12e3b232068caa13c1fd052c21b231a94e8c0a40bac25629f88"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Version:        1,
							SerializedSize: 77,
							TimeRange:      0,
							Inputs: []*types.TxInput{
								types.NewCoinbaseInput(testutil.MustDecodeHexString("0033")),
							},
							Outputs: []*types.TxOutput{
								types.NewTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab")),
							},
						}),
					},
				},
				verifyResult: []*bc.TxVerifyResult{{StatusFail: false}},
			},
		},
	}

	mustPostProcessBlock()
}

func TestProcessBlock(t *testing.T) {
	cases := []*processBlockTestCase{
		{
			desc: "process a invalid block",
			newBlock: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					Version:           1,
					Bits:              2305843009214532812,
					PreviousBlockHash: blockMap[0][0].block.Hash(),
				},
			},
			wantStore: createStoreItems([]int{0}, []*attachBlock{blockMap[0][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantError:        true,
		},
		{
			desc:      "process a orphan block normally",
			newBlock:  blockMap[2][0].block,
			wantStore: createStoreItems([]int{0}, []*attachBlock{blockMap[0][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil),
				},
				[]*state.BlockNode{
					mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil),
				},
			),
			wantOrphanManage: protocol.NewOrphanManageWithData(
				map[bc.Hash]*protocol.OrphanBlock{blockMap[2][0].block.Hash(): {Block: blockMap[2][0].block}},
				map[bc.Hash][]*bc.Hash{blockMap[2][0].block.PreviousBlockHash: {hashPtr(blockMap[2][0].block.Hash())}},
			),
			wantIsOrphan: true,
			wantError:    false,
		},
		{
			desc:      "attach a block normally",
			newBlock:  blockMap[1][0].block,
			wantStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil),
					mustNewBlockNode(&blockMap[1][0].block.BlockHeader, mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil)),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:      "init genesis block from db",
			newBlock:  blockMap[1][0].block,
			initStore: createStoreItems([]int{0}, []*attachBlock{blockMap[0][0]}),
			wantStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil),
					mustNewBlockNode(&blockMap[1][0].block.BlockHeader, mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil)),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:      "attach a block to fork chain normally, not rollback",
			newBlock:  blockMap[2][0].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[1][1]}),
			wantStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[1][1], blockMap[2][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[1][1].block.Hash(): mustCreateBlockNode(&blockMap[1][1].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "attach a block with btm transaction normally",
			newBlock: blockMap[2][1].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][1]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][1].block.Hash(): mustCreateBlockNode(&blockMap[2][1].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][1].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "attach a block with retire transaction normally",
			newBlock: blockMap[2][2].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][2]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][2].block.Hash(): mustCreateBlockNode(&blockMap[2][2].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][2].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "attach a block with issuance transaction normally",
			newBlock: blockMap[2][3].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][3]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][3].block.Hash(): mustCreateBlockNode(&blockMap[2][3].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][3].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "attach a block with issuance transaction but status fail is true",
			newBlock: blockMap[2][4].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][4]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][4].block.Hash(): mustCreateBlockNode(&blockMap[2][4].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][4].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "attach a block with non btm transaction",
			newBlock: blockMap[2][5].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][5]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][5].block.Hash(): mustCreateBlockNode(&blockMap[2][5].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][5].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "attach a block with non btm transaction but status fail is true",
			newBlock: blockMap[2][6].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][6]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][6].block.Hash(): mustCreateBlockNode(&blockMap[2][6].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][6].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:      "rollback a block only has coinbase transaction",
			newBlock:  blockMap[2][0].block,
			initStore: createStoreItems([]int{0, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[1][1]}),
			wantStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[1][1], blockMap[2][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[1][1].block.Hash(): mustCreateBlockNode(&blockMap[1][1].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has spend btm transaction",
			newBlock: blockMap[3][0].block,
			initStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][1]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][1], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][1].block.Hash(): mustCreateBlockNode(&blockMap[2][1].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has issuance transaction",
			newBlock: blockMap[3][0].block,
			initStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][3]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][3], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][3].block.Hash(): mustCreateBlockNode(&blockMap[2][3].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has issuance transaction but status fail is true",
			newBlock: blockMap[3][0].block,
			initStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][4]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][4], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][4].block.Hash(): mustCreateBlockNode(&blockMap[2][4].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has spend non btm",
			newBlock: blockMap[3][0].block,
			initStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][5]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][5], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][5].block.Hash(): mustCreateBlockNode(&blockMap[2][5].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has spend non btm but status fail is true",
			newBlock: blockMap[3][0].block,
			initStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][6]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][6], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][6].block.Hash(): mustCreateBlockNode(&blockMap[2][6].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:      "rollback a block only has coinbase tx, and from orphan manage",
			newBlock:  blockMap[1][0].block,
			initStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][1]}),
			initOrphanManage: protocol.NewOrphanManageWithData(
				map[bc.Hash]*protocol.OrphanBlock{
					blockMap[2][0].block.Hash(): protocol.NewOrphanBlock(blockMap[2][0].block, time.Now().Add(time.Minute*60)),
				},
				map[bc.Hash][]*bc.Hash{blockMap[2][0].block.PreviousBlockHash: {hashPtr(blockMap[2][0].block.Hash())}},
			),
			wantStore: createStoreItems([]int{0, 1, 3}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[1][1], blockMap[2][0]}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[1][1].block.Hash(): mustCreateBlockNode(&blockMap[1][1].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has spend btm tx, and from orphan manage",
			newBlock: blockMap[2][0].block,
			initStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][1]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			initOrphanManage: protocol.NewOrphanManageWithData(
				map[bc.Hash]*protocol.OrphanBlock{
					blockMap[3][0].block.Hash(): protocol.NewOrphanBlock(blockMap[3][0].block, time.Now().Add(time.Minute*60)),
				},
				map[bc.Hash][]*bc.Hash{blockMap[3][0].block.PreviousBlockHash: {hashPtr(blockMap[3][0].block.Hash())}},
			),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][1], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][1].block.Hash(): mustCreateBlockNode(&blockMap[2][1].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has retire tx, and from orphan manage",
			newBlock: blockMap[2][0].block,
			initStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][2]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			initOrphanManage: protocol.NewOrphanManageWithData(
				map[bc.Hash]*protocol.OrphanBlock{
					blockMap[3][0].block.Hash(): protocol.NewOrphanBlock(blockMap[3][0].block, time.Now().Add(time.Minute*60)),
				},
				map[bc.Hash][]*bc.Hash{blockMap[3][0].block.PreviousBlockHash: {hashPtr(blockMap[3][0].block.Hash())}},
			),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][2], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][2].block.Hash(): mustCreateBlockNode(&blockMap[2][2].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has issuance tx, and from orphan manage",
			newBlock: blockMap[2][0].block,
			initStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][3]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			initOrphanManage: protocol.NewOrphanManageWithData(
				map[bc.Hash]*protocol.OrphanBlock{
					blockMap[3][0].block.Hash(): protocol.NewOrphanBlock(blockMap[3][0].block, time.Now().Add(time.Minute*60)),
				},
				map[bc.Hash][]*bc.Hash{blockMap[3][0].block.PreviousBlockHash: {hashPtr(blockMap[3][0].block.Hash())}},
			),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][3], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][3].block.Hash(): mustCreateBlockNode(&blockMap[2][3].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
		{
			desc:     "rollback a block has non btm tx, and from orphan manage",
			newBlock: blockMap[2][0].block,
			initStore: createStoreItems([]int{0, 1, 2}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][5]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 1, Spent: false},
			}),
			initOrphanManage: protocol.NewOrphanManageWithData(
				map[bc.Hash]*protocol.OrphanBlock{
					blockMap[3][0].block.Hash(): protocol.NewOrphanBlock(blockMap[3][0].block, time.Now().Add(time.Minute*60)),
				},
				map[bc.Hash][]*bc.Hash{blockMap[3][0].block.PreviousBlockHash: {hashPtr(blockMap[3][0].block.Hash())}},
			),
			wantStore: createStoreItems([]int{0, 1, 2, 4}, []*attachBlock{blockMap[0][0], blockMap[1][0], blockMap[2][0], blockMap[2][5], blockMap[3][0]}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("c93b687f98d039046cd2afd514c62f5d1c2c3b0804e4845b00a33e736ef48a33"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}, &storeItem{
				key: database.CalcUtxoKey(hashPtr(testutil.MustDecodeHash("be164edbce8bcd1d890c1164541b8418fdcb257499757d3b88561bca06e97e29"))),
				val: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 0, Spent: false},
			}),
			wantBlockIndex: state.NewBlockIndexWithData(
				map[bc.Hash]*state.BlockNode{
					blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][0].block.Hash(): mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[2][5].block.Hash(): mustCreateBlockNode(&blockMap[2][5].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					blockMap[3][0].block.Hash(): mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
				[]*state.BlockNode{
					mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
					mustCreateBlockNode(&blockMap[3][0].block.BlockHeader, &blockMap[2][0].block.BlockHeader, &blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantIsOrphan:     false,
			wantError:        false,
		},
	}

	for _, c := range cases {
		if err := c.Run(); err != nil {
			panic(err)
		}
	}
}

func createStoreItems(mainChainIndexes []int, attachBlocks []*attachBlock, extralItem ...*storeItem) storeItems {
	var items storeItems
	for _, item := range extralItem {
		items = append(items, item)
	}

	mainChainIndexMap := make(map[int]interface{})
	for _, index := range mainChainIndexes {
		mainChainIndexMap[index] = nil
	}

	for i, attachBlock := range attachBlocks {
		block := attachBlock.block
		blockHash := block.Hash()
		items = append(items, &storeItem{
			key: database.CalcBlockKey(&blockHash),
			val: block,
		})

		items = append(items, &storeItem{
			key: database.CalcTxStatusKey(&blockHash),
			val: &bc.TransactionStatus{Version: 1, VerifyStatus: attachBlock.verifyResult},
		})
		items = append(items, &storeItem{
			key: database.CalcBlockHeaderKey(block.Height, &blockHash),
			val: block.BlockHeader,
		})

		if _, ok := mainChainIndexMap[i]; !ok {
			continue
		}

		for i, tx := range block.Transactions {
			statusFail := attachBlock.verifyResult[i].StatusFail
			for _, input := range tx.Inputs {
				if statusFail && input.AssetID() != *consensus.BTMAssetID {
					continue
				}

				if _, ok := input.TypedInput.(*types.SpendInput); !ok {
					continue
				}
				spendOutputID, err := input.SpentOutputID()
				if err != nil {
					panic(err)
				}
				index := spendUTXO(spendOutputID, items, block.Height)
				items = append(items[0:index], items[index+1:]...)
			}
			for j, output := range tx.Outputs {
				if statusFail && *tx.Outputs[j].AssetId != *consensus.BTMAssetID {
					continue
				}
				if output.ControlProgram[0] == 0x6a {
					continue
				}
				items = append(items, &storeItem{key: database.CalcUtxoKey(tx.Tx.ResultIds[j]),
					val: &storage.UtxoEntry{IsCoinBase: i == 0, BlockHeight: block.Height, Spent: false},
				})
			}
		}
	}

	lastIndex := mainChainIndexes[len(mainChainIndexes)-1]
	betBlock := attachBlocks[lastIndex].block
	bestBlockHash := betBlock.Hash()
	items = append(items, &storeItem{
		key: database.BlockStoreKey,
		val: &protocol.BlockStoreState{Height: betBlock.Height, Hash: &bestBlockHash},
	})
	return items
}

func hashPtr(hash bc.Hash) *bc.Hash {
	return &hash
}

func mustCreateBlockNode(header *types.BlockHeader, parents ...*types.BlockHeader) *state.BlockNode {
	var parentNode *state.BlockNode
	for i := len(parents) - 1; i >= 0; i-- {
		parentNode = mustNewBlockNode(parents[i], parentNode)
	}
	return mustNewBlockNode(header, parentNode)
}

func mustPostProcessBlock() {
	for _, blocks := range blockMap {
		for _, attachBlock := range blocks {
			mustCalcMerkleRootHash(attachBlock)
			mustFillTransactionSize(attachBlock.block)
			sortSpendOutputID(attachBlock.block)
		}
	}
}

func mustCalcMerkleRootHash(attachBlock *attachBlock) {
	bcBlock := types.MapBlock(attachBlock.block)
	txStatusHash, err := types.TxStatusMerkleRoot(attachBlock.verifyResult)
	if err != nil {
		panic("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot(bcBlock.Transactions)
	if err != nil {
		panic("fail on calc genesis tx merkel root")
	}

	attachBlock.block.TransactionStatusHash = txStatusHash
	attachBlock.block.TransactionsMerkleRoot = merkleRoot
}

func mustFillTransactionSize(block *types.Block) {
	for _, tx := range block.Transactions {
		bytes, err := tx.MarshalText()
		if err != nil {
			panic(err)
		}
		tx.TxData.SerializedSize = uint64(len(bytes) / 2)
		tx.Tx.SerializedSize = uint64(len(bytes) / 2)
	}
}

func mustNewBlockNode(h *types.BlockHeader, parent *state.BlockNode) *state.BlockNode {
	node, err := state.NewBlockNode(h, parent)
	if err != nil {
		panic(err)
	}
	return node
}

func spendUTXO(spendOutputID bc.Hash, items storeItems, blockHeight uint64) int {
	for i, item := range items {
		utxo, ok := item.val.(*storage.UtxoEntry)
		if !ok {
			continue
		}
		if string(database.CalcUtxoKey(&spendOutputID)) != string(item.key) {
			continue
		}
		if utxo.Spent || (utxo.IsCoinBase && utxo.BlockHeight+consensus.CoinbasePendingBlockNumber > blockHeight) {
			panic("utxo can not be use")
		}
		utxo.Spent = true
		return i
	}
	panic("can not find available utxo")
}
