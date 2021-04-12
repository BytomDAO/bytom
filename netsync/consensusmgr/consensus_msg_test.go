package consensusmgr

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/tendermint/go-wire"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

var _ = wire.RegisterInterface(
	struct{ ConsensusMessage }{},
	wire.ConcreteType{O: &BlockSignatureMsg{}, Byte: blockSignatureByte},
	wire.ConcreteType{O: &BlockProposeMsg{}, Byte: blockProposeByte},
)

func TestDecodeMessage(t *testing.T) {
	testCases := []struct {
		msg     ConsensusMessage
		msgType byte
	}{
		{
			msg: &BlockSignatureMsg{
				BlockHash: [32]byte{0x01},
				Signature: []byte{0x00},
				PubKey:    []byte{0x01},
			},
			msgType: blockSignatureByte,
		},
		{
			msg: &BlockProposeMsg{
				RawBlock: []byte{0x01, 0x02},
			},
			msgType: blockProposeByte,
		},
	}
	for i, c := range testCases {
		binMsg := wire.BinaryBytes(struct{ ConsensusMessage }{c.msg})
		gotMsgType, gotMsg, err := decodeMessage(binMsg)
		if err != nil {
			t.Fatalf("index:%d decode Message err %s", i, err)
		}
		if gotMsgType != c.msgType {
			t.Fatalf("index:%d decode Message type err. got:%d want:%d", i, gotMsgType, c.msg)
		}
		if !reflect.DeepEqual(gotMsg, c.msg) {
			t.Fatalf("index:%d decode Message err. got:%s\n want:%s", i, spew.Sdump(gotMsg), spew.Sdump(c.msg))
		}
	}
}

func TestBlockSignBroadcastMsg(t *testing.T) {
	blockSignMsg := &BlockSignatureMsg{
		BlockHash: [32]byte{0x01},
		Signature: []byte{0x00},
		PubKey:    []byte{0x01},
	}
	signatureBroadcastMsg := NewBroadcastMsg(NewBlockSignatureMsg(bc.NewHash(blockSignMsg.BlockHash), blockSignMsg.Signature, blockSignMsg.PubKey), consensusChannel)

	binMsg := wire.BinaryBytes(signatureBroadcastMsg.GetMsg())
	gotMsgType, gotMsg, err := decodeMessage(binMsg)
	if err != nil {
		t.Fatalf("decode Message err %s", err)
	}
	if gotMsgType != blockSignatureByte {
		t.Fatalf("decode Message type err. got:%d want:%d", gotMsgType, blockSignatureByte)
	}
	if !reflect.DeepEqual(gotMsg, blockSignMsg) {
		t.Fatalf("decode Message err. got:%s\n want:%s", spew.Sdump(gotMsg), spew.Sdump(blockSignMsg))
	}
}

func TestBlockProposeBroadcastMsg(t *testing.T) {
	blockProposeMsg, _ := NewBlockProposeMsg(testBlock)

	proposeBroadcastMsg := NewBroadcastMsg(blockProposeMsg, consensusChannel)

	binMsg := wire.BinaryBytes(proposeBroadcastMsg.GetMsg())
	gotMsgType, gotMsg, err := decodeMessage(binMsg)
	if err != nil {
		t.Fatalf("decode Message err %s", err)
	}
	if gotMsgType != blockProposeByte {
		t.Fatalf("decode Message type err. got:%d want:%d", gotMsgType, blockProposeByte)
	}
	if !reflect.DeepEqual(gotMsg, blockProposeMsg) {
		t.Fatalf("decode Message err. got:%s\n want:%s", spew.Sdump(gotMsg), spew.Sdump(blockProposeMsg))
	}
}

var testBlock = &types.Block{
	BlockHeader: types.BlockHeader{
		Version:   1,
		Height:    0,
		Timestamp: 1528945000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
		},
	},
}

func TestBlockProposeMsg(t *testing.T) {
	blockMsg, err := NewBlockProposeMsg(testBlock)
	if err != nil {
		t.Fatalf("create new mine block msg err:%s", err)
	}

	gotBlock, err := blockMsg.(*BlockProposeMsg).GetProposeBlock()
	if err != nil {
		t.Fatalf("got block err:%s", err)
	}

	if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
		t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
	}

	wantString := "{block_height: 0, block_hash: 3ce98dfffbd0e10c318f167696603b23173b3ec86e7868c8fa65be76edefc67e}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}

	blockMsg.(*BlockProposeMsg).RawBlock[1] = blockMsg.(*BlockProposeMsg).RawBlock[1] + 0x1
	_, err = blockMsg.(*BlockProposeMsg).GetProposeBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}

	wantString = "{err: wrong message}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}
}

func TestBlockSignatureMsg(t *testing.T) {
	msg := &BlockSignatureMsg{
		BlockHash: [32]byte{0x01},
		Signature: []byte{0x00},
		PubKey:    []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	gotMsg := NewBlockSignatureMsg(bc.NewHash(msg.BlockHash), msg.Signature, msg.PubKey)

	if !reflect.DeepEqual(gotMsg, msg) {
		t.Fatalf("test block signature message err. got:%s\n want:%s", spew.Sdump(gotMsg), spew.Sdump(msg))
	}
	wantString := "{block_hash: 0100000000000000000000000000000000000000000000000000000000000000,signature:00,pubkey:01000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000}"
	if gotMsg.String() != wantString {
		t.Fatalf("test block signature message err. got string:%s\n want string:%s", gotMsg.String(), wantString)
	}
}
