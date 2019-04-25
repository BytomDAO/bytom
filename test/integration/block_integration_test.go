package integration

import (
	"testing"

	"github.com/bytom/config"
	"github.com/bytom/database"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

func TestProcessBlock(t *testing.T) {
	gensisBlock := config.GenesisBlock()
	genesisBlockHash := gensisBlock.Hash()
	fillTransactionSize(gensisBlock)

	cases := []*processBlockTestCase{
		{
			desc: "process a invalid block",
			newBlock: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					Version:           1,
					PreviousBlockHash: genesisBlockHash,
				},
			},
			wantStore: storeItems{
				{
					key: database.BlockStoreKey,
					val: &protocol.BlockStoreState{Height: 0, Hash: &genesisBlockHash},
				},
				{
					key: database.CalcBlockKey(&genesisBlockHash),
					val: gensisBlock,
				},
				{
					key: database.CalcTxStatusKey(&genesisBlockHash),
					val: &bc.TransactionStatus{Version: 1, VerifyStatus: []*bc.TxVerifyResult{{StatusFail: false}}},
				},
				{
					key: database.CalcBlockHeaderKey(gensisBlock.Height, &genesisBlockHash),
					val: gensisBlock.BlockHeader,
				},
				{
					key: database.CalcUtxoKey(gensisBlock.Transactions[0].Tx.ResultIds[0]),
					val: &storage.UtxoEntry{IsCoinBase: true, BlockHeight: 0, Spent: false},
				},
			},
			wantBlockIndex: state.NewBlockIndexWithInitData(
				map[bc.Hash]*state.BlockNode{
					genesisBlockHash: mustNewBlockNode(&gensisBlock.BlockHeader, nil),
				},
				[]*state.BlockNode{
					mustNewBlockNode(&gensisBlock.BlockHeader, nil),
				},
			),
			wantOrphanManage: protocol.NewOrphanManage(),
			wantError:        true,
		},
	}

	for _, c := range cases {
		if err := c.Run(); err != nil {
			panic(err)
		}
	}
}

func mustNewBlockNode(h *types.BlockHeader, parent *state.BlockNode) *state.BlockNode {
	node, err := state.NewBlockNode(h, parent)
	if err != nil {
		panic(err)
	}
	return node
}

func fillTransactionSize(block *types.Block) {
	for _, tx := range block.Transactions {
		bytes, err := tx.MarshalText()
		if err != nil {
			panic(err)
		}
		tx.TxData.SerializedSize = uint64(len(bytes) / 2)
		tx.Tx.SerializedSize = uint64(len(bytes) / 2)
	}
}
