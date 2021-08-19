package integration

import (
	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/database"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/testutil"
)

var blockMap map[int][]*attachBlock

type attachBlock struct {
	block *types.Block
}

func init() {
	consensus.ActiveNetParams = consensus.SoloNetParams

	blockMap = map[int][]*attachBlock{
		0: {
			{
				block: config.GenesisBlock(),
			},
		},
		// 0 号的hash不会变
		1: {
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            1,
						Version:           1,
						Timestamp:         1556431597,
						PreviousBlockHash: testutil.MustDecodeHash("6f62777fab457d134aa55d29197ea5874988627d8777f6a14ed032a2f06dcc2f"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            1,
						Version:           1,
						Timestamp:         1556431697,
						PreviousBlockHash: testutil.MustDecodeHash("6f62777fab457d134aa55d29197ea5874988627d8777f6a14ed032a2f06dcc2f"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("00143d05e891b165b165afefa2e861e83a9745f80d8c"), [][]byte{}),
							},
						}),
					},
				},
			},
		},
		2: {
			//the below blocks's previous block is blockMap[1][0]
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431604,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
			// with spend btm transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431604,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
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
									[][]byte{},
								),
							},
							Outputs: []*types.TxOutput{
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("00143d05e891b165b165afefa2e861e83a9745f80d8c"), [][]byte{}),
							},
						}),
					},
				},
			},
			// with btm retire transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
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
									[][]byte{},
								),
							},
							Outputs: []*types.TxOutput{
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("6a"), [][]byte{}), // retire
							},
						}),
					},
				},
			},
			// with issuance transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
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
									[][]byte{},
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
								types.NewOriginalTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
			// with issuance transaction but status fail is true
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
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
									[][]byte{},
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
								types.NewOriginalTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
			// with non btm transaction
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
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
									[][]byte{},
								),
								types.NewSpendInput(
									[][]byte{
										testutil.MustDecodeHexString("cd6fb451102db667341438f20dbeabd30b343ed08d89625a8e27e82478e89ddea9e7d51f8a4036e0cc2602ac5fae0bdbfda025a0e2c12e3ddc8100b62461670b"),
										testutil.MustDecodeHexString("33b05e00e19cb2bdbc8a6a67b4f1e03fc265534bcfc7641b305c8204fb486f79"),
									},
									testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f22"),
									testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, 1,
									testutil.MustDecodeHexString("0014cade6dd7cbe2ea2b8ab90dfb8756dda4ba1624bc"),
									[][]byte{},
								),
							},
							Outputs: []*types.TxOutput{
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
								types.NewOriginalTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
			// with non btm transaction but status fail is true
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            2,
						Version:           1,
						Timestamp:         1556431607,
						PreviousBlockHash: testutil.MustDecodeHash("0311998e27abc1c2f5cc1f86b1aca5e7dd3ca65d63359e0c7539c40207923e10"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41350000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
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
									[][]byte{},
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
									[][]byte{},
								),
							},
							Outputs: []*types.TxOutput{
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 9900000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
								types.NewOriginalTxOutput(testutil.MustDecodeAsset("641ccb49dd38df9921a55e020d40a2323589c36ab5557f8a249ee01cc09d1836"), 10000000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
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
						PreviousBlockHash: testutil.MustDecodeHash("d96091fb7784af594980012cadb05ad717d45603eab2b2105a2735ae5bb3aca3"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
			// the previous block is blockMap[2][2]
			{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height:            3,
						Version:           1,
						Timestamp:         1556431640,
						PreviousBlockHash: testutil.MustDecodeHash("222356873e67eacf3fa38ddb736c467471c8aa91f65686d28a96f7d39f8278e7"),
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
								types.NewOriginalTxOutput(*consensus.BTMAssetID, 41250000000, testutil.MustDecodeHexString("0014024bb9bfc639bdac292ff9ceb41b5c6f5a970eab"), [][]byte{}),
							},
						}),
					},
				},
			},
		},
	}

	mustPostProcessBlock()
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
			val: block,
		})
		items = append(items, &storeItem{
			key: database.CalcBlockHeaderKey(&blockHash),
			val: block.BlockHeader,
		})
		if _, ok := mainChainIndexMap[i]; !ok {
			continue
		}

		for i, tx := range block.Transactions {
			for _, input := range tx.Inputs {
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
				if output.ControlProgram[0] == 0x6a {
					continue
				}
				utxoType := storage.NormalUTXOType
				if i == 0 {
					utxoType = storage.CoinbaseUTXOType
				}
				items = append(items, &storeItem{key: database.CalcUtxoKey(tx.Tx.ResultIds[j]),
					val: &storage.UtxoEntry{Type: utxoType, BlockHeight: block.Height, Spent: false},
				})
			}
		}
	}

	lastIndex := mainChainIndexes[len(mainChainIndexes)-1]
	betBlock := attachBlocks[lastIndex].block
	bestBlockHash := betBlock.Hash()
	items = append(items, &storeItem{
		key: database.BlockStoreKey,
		val: &state.BlockStoreState{Height: betBlock.Height, Hash: &bestBlockHash},
	})
	return items
}

func hashPtr(hash bc.Hash) *bc.Hash {
	return &hash
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
	merkleRoot, err := types.TxMerkleRoot(bcBlock.Transactions)
	if err != nil {
		panic("fail on calc genesis tx merkel root")
	}

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

func spendUTXO(spendOutputID bc.Hash, items storeItems, blockHeight uint64) int {
	for i, item := range items {
		utxo, ok := item.val.(*storage.UtxoEntry)
		if !ok {
			continue
		}
		if string(database.CalcUtxoKey(&spendOutputID)) != string(item.key) {
			continue
		}
		if utxo.Spent || (utxo.Type == storage.CoinbaseUTXOType && utxo.BlockHeight+consensus.CoinbasePendingBlockNumber > blockHeight) {
			panic("utxo can not be use")
		}
		utxo.Spent = true
		return i
	}
	panic("can not find available utxo")
}

func createStoreEntries(mainChainIndexes []int, attachBlocks []*attachBlock, extralItem ...*storeItem) []storeEntry {
	storeItems := createStoreItems(mainChainIndexes, attachBlocks, extralItem...)
	var storeEntries []storeEntry
	for _, item := range storeItems {
		entrys, err := serialItem(item)
		if err != nil {
			panic(err)
		}

		storeEntries = append(storeEntries, entrys...)
	}

	return storeEntries
}
