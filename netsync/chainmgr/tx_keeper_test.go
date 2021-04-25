package chainmgr

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/bytom/consensus"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol"
	core "github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/test/mock"
	"github.com/bytom/bytom/testcontrol"
)

const txsNumber = 2000

type mempool struct {
}

func (m *mempool) GetTransactions() []*core.TxDesc {
	txs := []*core.TxDesc{}
	for i := 0; i < txsNumber; i++ {
		txInput := types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, uint64(i), 1, []byte{0x51}, [][]byte{})
		txInput.CommitmentSuffix = []byte{0, 1, 2}
		txInput.WitnessSuffix = []byte{0, 1, 2}

		tx := &types.Tx{

			TxData: types.TxData{
				//SerializedSize: uint64(i * 10),
				Inputs: []*types.TxInput{
					txInput,
				},
				Outputs: []*types.TxOutput{
					types.NewOriginalTxOutput(*consensus.BTMAssetID, uint64(i), []byte{0x6a}, [][]byte{}),
				},
				SerializedSize: 1000,
			},
			Tx: &bc.Tx{
				ID: bc.Hash{V0: uint64(i), V1: uint64(i), V2: uint64(i), V3: uint64(i)},
			},
		}
		txs = append(txs, &core.TxDesc{Tx: tx})
	}
	return txs
}

func (m *mempool) IsDust(tx *types.Tx) bool {
	return false
}

func TestSyncMempool(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	tmpDir, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	testDBA := dbm.NewDB("testdba", "leveldb", tmpDir)
	testDBB := dbm.NewDB("testdbb", "leveldb", tmpDir)

	blocks := mockBlocks(nil, 5)
	a := mockSync(blocks, &mock.Mempool{}, testDBA)
	b := mockSync(blocks, &mock.Mempool{}, testDBB)
	a.mempool = &mempool{}
	netWork := NewNetWork()
	netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
	netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
	if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
		t.Errorf("fail on peer hands shake %v", err)
	} else {
		go B2A.postMan()
		go A2B.postMan()
	}

	go a.syncMempoolLoop()
	a.syncMempool("test node B")
	wantTxs := a.mempool.GetTransactions()

	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	gotTxs := []*protocol.TxDesc{}
	for {
		select {
		case <-ticker.C:
			gotTxs = b.mempool.GetTransactions()
			if len(gotTxs) >= txsNumber {
				goto out
			}
		case <-timeout.C:
			t.Fatalf("mempool sync timeout")
		}
	}

out:
	if len(gotTxs) != txsNumber {
		t.Fatalf("mempool sync txs num err. got:%d want:%d", len(gotTxs), txsNumber)
	}

	for i, gotTx := range gotTxs {
		index := gotTx.Tx.Inputs[0].Amount()
		if !reflect.DeepEqual(gotTx.Tx.Inputs[0].Amount(), wantTxs[index].Tx.Inputs[0].Amount()) {
			t.Fatalf("mempool tx err. index:%d\n,gotTx:%s\n,wantTx:%s", i, spew.Sdump(gotTx.Tx.Inputs), spew.Sdump(wantTxs[0].Tx.Inputs))
		}

		if !reflect.DeepEqual(gotTx.Tx.Outputs[0].AssetAmount, wantTxs[index].Tx.Outputs[0].AssetAmount) {
			t.Fatalf("mempool tx err. index:%d\n,gotTx:%s\n,wantTx:%s", i, spew.Sdump(gotTx.Tx.Outputs), spew.Sdump(wantTxs[0].Tx.Outputs))
		}
	}
}

func TestBroadcastTxsLoop(t *testing.T) {
	if testcontrol.IgnoreTestTemporary {
		return
	}

	tmpDir, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	testDBA := dbm.NewDB("testdba", "leveldb", tmpDir)
	testDBB := dbm.NewDB("testdbb", "leveldb", tmpDir)

	blocks := mockBlocks(nil, 5)
	a := mockSync(blocks, &mock.Mempool{}, testDBA)
	b := mockSync(blocks, &mock.Mempool{}, testDBB)
	a.mempool = &mempool{}
	netWork := NewNetWork()
	netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
	netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
	if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
		t.Errorf("fail on peer hands shake %v", err)
	} else {
		go B2A.postMan()
		go A2B.postMan()
	}
	a.txMsgSub, err = a.eventDispatcher.Subscribe(core.TxMsgEvent{})
	if err != nil {
		t.Fatal("txMsgSub subscribe err", err)
	}
	go a.broadcastTxsLoop()
	wantTxs := a.mempool.GetTransactions()
	txsNum := 50
	for i, txD := range wantTxs {
		if i >= txsNum {
			break
		}
		a.eventDispatcher.Post(core.TxMsgEvent{TxMsg: &core.TxPoolMsg{TxDesc: txD, MsgType: core.MsgNewTx}})
	}
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	gotTxs := []*protocol.TxDesc{}
	for {
		select {
		case <-ticker.C:
			gotTxs = b.mempool.GetTransactions()
			if len(gotTxs) >= txsNum {
				goto out
			}
		case <-timeout.C:
			t.Fatalf("mempool sync timeout")
		}
	}

out:
	if len(gotTxs) != txsNum {
		t.Fatalf("mempool sync txs num err. got:%d want:%d", len(gotTxs), txsNumber)
	}

	for i, gotTx := range gotTxs {
		index := gotTx.Tx.Inputs[0].Amount()
		if !reflect.DeepEqual(gotTx.Tx.Inputs[0].Amount(), wantTxs[index].Tx.Inputs[0].Amount()) {
			t.Fatalf("mempool tx err. index:%d\n,gotTx:%s\n,wantTx:%s", i, spew.Sdump(gotTx.Tx.Inputs), spew.Sdump(wantTxs[0].Tx.Inputs))
		}

		if !reflect.DeepEqual(gotTx.Tx.Outputs[0].AssetAmount, wantTxs[index].Tx.Outputs[0].AssetAmount) {
			t.Fatalf("mempool tx err. index:%d\n,gotTx:%s\n,wantTx:%s", i, spew.Sdump(gotTx.Tx.Outputs), spew.Sdump(wantTxs[0].Tx.Outputs))
		}
	}
}
