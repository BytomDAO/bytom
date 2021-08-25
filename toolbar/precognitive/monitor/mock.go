package monitor

import (
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/test/mock"
)

func mockChainAndPool() (*mock.Chain, *mock.Mempool, error) {
	txPool := &mock.Mempool{}
	mockChain := mock.NewChain(txPool)
	genesisBlock, err := getGenesisBlock()
	if err != nil {
		return nil, nil, err
	}

	mockChain.SetBlockByHeight(genesisBlock.BlockHeader.Height, genesisBlock)
	mockChain.SetBestBlockHeader(&genesisBlock.BlockHeader)
	return mockChain, txPool, nil
}

func getGenesisBlock() (*types.Block, error) {
	genesisBlock := &types.Block{}
	if err := genesisBlock.UnmarshalText([]byte("030100000000000000000000000000000000000000000000000000000000000000000082bfe3f4bf2d4052415e796436f587fac94677b20f027e910b70e2c220c411c0e87c37e0e1cc2ec9c377e5192668bc0a367e4a4764f11e7c725ecced1d7b6a492974fab1b6d5bc01000107010001012402220020f86826d640810eb08a2bfb706e0092273e05e9a7d3d71f9d53f4f6cc2e3d6c6a0001013b0039ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00011600148c9d063ff74ee6d9ffa88d83aeb038068366c4c400")); err != nil {
		return nil, err
	}

	return genesisBlock, nil
}
