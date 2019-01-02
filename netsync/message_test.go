package netsync

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

var txs = []*types.Tx{
	types.NewTx(types.TxData{
		SerializedSize: uint64(50),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
		Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 5000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(51),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02})},
		Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 5000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(52),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02, 0x03})},
		Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 5000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(52),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02, 0x03})},
		Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 2000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(52),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02, 0x03})},
		Outputs:        []*types.TxOutput{types.NewTxOutput(*consensus.BTMAssetID, 10000, nil)},
	}),
}

func TestTransactionMessage(t *testing.T) {
	for _, tx := range txs {
		txMsg, err := NewTransactionMessage(tx)
		if err != nil {
			t.Fatalf("create tx msg err:%s", err)
		}

		gotTx, err := txMsg.GetTransaction()
		if err != nil {
			t.Fatalf("get txs from txsMsg err:%s", err)
		}
		if !reflect.DeepEqual(*tx.Tx, *gotTx.Tx) {
			t.Errorf("txs msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(gotTx.Tx))
		}
	}
}

func TestTransactionsMessage(t *testing.T) {
	txsMsg, err := NewTransactionsMessage(txs)
	if err != nil {
		t.Fatalf("create txs msg err:%s", err)
	}

	gotTxs, err := txsMsg.GetTransactions()
	if err != nil {
		t.Fatalf("get txs from txsMsg err:%s", err)
	}

	if len(gotTxs) != len(txs) {
		t.Fatal("txs msg test err: number of txs not match ")
	}

	for i, tx := range txs {
		if !reflect.DeepEqual(tx.Tx, gotTxs[i].Tx) {
			t.Errorf("txs msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(gotTxs[i].Tx))
		}
	}
}

var testBlock = &types.Block{
	BlockHeader: types.BlockHeader{
		Version:   1,
		Height:    0,
		Nonce:     9253507043297,
		Timestamp: 1528945000,
		Bits:      2305843009214532812,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	Transactions: txs,
}

func TestBlockMessage(t *testing.T) {
	blockMsg, err := NewBlockMessage(testBlock)
	if err != nil {
		t.Fatalf("create new block msg err:%s", err)
	}

	gotBlock, err := blockMsg.GetBlock()
	if err != nil {
		t.Fatalf("got block err:%s", err)
	}

	if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
		t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
	}

	for i, tx := range gotBlock.Transactions {
		if !reflect.DeepEqual(tx.Tx, txs[i].Tx) {
			t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(txs[i].Tx))
		}
	}

	blockMsg.RawBlock[1] = blockMsg.RawBlock[1] + 0x1
	_, err = blockMsg.GetBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}
}

func TestMinedBlockMessage(t *testing.T) {
	blockMsg, err := NewMinedBlockMessage(testBlock)
	if err != nil {
		t.Fatalf("create new block msg err:%s", err)
	}

	gotBlock, err := blockMsg.GetMineBlock()
	if err != nil {
		t.Fatalf("got block err:%s", err)
	}

	if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
		t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
	}

	for i, tx := range gotBlock.Transactions {
		if !reflect.DeepEqual(tx.Tx, txs[i].Tx) {
			t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(txs[i].Tx))
		}
	}

	blockMsg.RawBlock[1] = blockMsg.RawBlock[1] + 0x1
	_, err = blockMsg.GetMineBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}
}

var testHeaders = []*types.BlockHeader{
	{
		Version:   1,
		Height:    0,
		Nonce:     9253507043297,
		Timestamp: 1528945000,
		Bits:      2305843009214532812,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	{
		Version:   1,
		Height:    1,
		Nonce:     9253507043298,
		Timestamp: 1528945000,
		Bits:      2305843009214532812,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	{
		Version:   1,
		Height:    3,
		Nonce:     9253507043298,
		Timestamp: 1528945000,
		Bits:      2305843009214532812,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
}

func TestHeadersMessage(t *testing.T) {
	headersMsg, err := NewHeadersMessage(testHeaders)
	if err != nil {
		t.Fatalf("create headers msg err:%s", err)
	}

	gotHeaders, err := headersMsg.GetHeaders()
	if err != nil {
		t.Fatalf("got headers err:%s", err)
	}

	if !reflect.DeepEqual(gotHeaders, testHeaders) {
		t.Errorf("headers msg test err: got %s\nwant %s", spew.Sdump(gotHeaders), spew.Sdump(testHeaders))
	}
}

func TestGetBlockMessage(t *testing.T) {
	getBlockMsg := GetBlockMessage{RawHash: [32]byte{0x01}}
	gotHash := getBlockMsg.GetHash()

	if !reflect.DeepEqual(gotHash.Byte32(), getBlockMsg.RawHash) {
		t.Errorf("get block msg test err: got %s\nwant %s", spew.Sdump(gotHash.Byte32()), spew.Sdump(getBlockMsg.RawHash))
	}
}

type testGetHeadersMessage struct {
	blockLocator []*bc.Hash
	stopHash     *bc.Hash
}

func TestGetHeadersMessage(t *testing.T) {
	testMsg := testGetHeadersMessage{
		blockLocator: []*bc.Hash{{V0: 0x01}, {V0: 0x02}, {V0: 0x03}},
		stopHash:     &bc.Hash{V0: 0xaa, V2: 0x55},
	}
	getHeadersMsg := NewGetHeadersMessage(testMsg.blockLocator, testMsg.stopHash)
	gotBlockLocator := getHeadersMsg.GetBlockLocator()
	gotStopHash := getHeadersMsg.GetStopHash()

	if !reflect.DeepEqual(testMsg.blockLocator, gotBlockLocator) {
		t.Errorf("get headers msg test err: got %s\nwant %s", spew.Sdump(gotBlockLocator), spew.Sdump(testMsg.blockLocator))
	}

	if !reflect.DeepEqual(testMsg.stopHash, gotStopHash) {
		t.Errorf("get headers msg test err: got %s\nwant %s", spew.Sdump(gotStopHash), spew.Sdump(testMsg.stopHash))
	}
}

var testBlocks = []*types.Block{
	{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     9253507043297,
			Timestamp: 1528945000,
			Bits:      2305843009214532812,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
				TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
			},
		},
		Transactions: txs,
	},
	{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     9253507043297,
			Timestamp: 1528945000,
			Bits:      2305843009214532812,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
				TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
			},
		},
		Transactions: txs,
	},
}

func TestBlocksMessage(t *testing.T) {
	blocksMsg, err := NewBlocksMessage(testBlocks)
	if err != nil {
		t.Fatalf("create blocks msg err:%s", err)
	}
	gotBlocks, err := blocksMsg.GetBlocks()
	if err != nil {
		t.Fatalf("get blocks err:%s", err)
	}

	for _, gotBlock := range gotBlocks {
		if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
			t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
		}

		for i, tx := range gotBlock.Transactions {
			if !reflect.DeepEqual(tx.Tx, txs[i].Tx) {
				t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(txs[i].Tx))
			}
		}
	}
}

func TestStatusResponseMessage(t *testing.T) {
	statusResponseMsg := NewStatusResponseMessage(&testBlock.BlockHeader)
	gotHash := statusResponseMsg.GetHash()
	if !reflect.DeepEqual(*gotHash, testBlock.Hash()) {
		t.Errorf("status response msg test err: got %s\nwant %s", spew.Sdump(*gotHash), spew.Sdump(testBlock.Hash()))
	}
}
