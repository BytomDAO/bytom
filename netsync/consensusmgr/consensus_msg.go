package consensusmgr

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	blockSignatureByte = byte(0x10)
	blockProposeByte   = byte(0x11)
)

// ConsensusMessage is a generic message for consensus reactor.
type ConsensusMessage interface {
	String() string
	BroadcastMarkSendRecord(ps *peers.PeerSet, peers []string)
	BroadcastFilterTargetPeers(ps *peers.PeerSet) []string
}

var _ = wire.RegisterInterface(
	struct{ ConsensusMessage }{},
	wire.ConcreteType{O: &BlockVerificationMsg{}, Byte: blockSignatureByte},
	wire.ConcreteType{O: &BlockProposeMsg{}, Byte: blockProposeByte},
)

// decodeMessage decode msg
func decodeMessage(bz []byte) (msgType byte, msg ConsensusMessage, err error) {
	msgType = bz[0]
	n := int(0)
	r := bytes.NewReader(bz)
	msg = wire.ReadBinary(struct{ ConsensusMessage }{}, r, maxBlockchainResponseSize, &n, &err).(struct{ ConsensusMessage }).ConsensusMessage
	if err != nil && n != len(bz) {
		err = errors.New("DecodeMessage() had bytes left over")
	}
	return
}

// BlockVerificationMsg block verification message transferred between nodes.
type BlockVerificationMsg struct {
	SourceHash bc.Hash
	TargetHash bc.Hash
	PubKey     []byte
	Signature  []byte
}

// NewBlockVerificationMsg create new block verification msg.
func NewBlockVerificationMsg(sourceHash, targetHash bc.Hash, pubKey, signature []byte) ConsensusMessage {
	return &BlockVerificationMsg{
		SourceHash: sourceHash,
		TargetHash: targetHash,
		PubKey:     pubKey,
		Signature:  signature,
	}
}

func (b *BlockVerificationMsg) String() string {
	return fmt.Sprintf("{sourceHash:%s,targetHash:%s,signature:%s,pubkey:%s}",
		b.SourceHash.String(), b.TargetHash.String(), hex.EncodeToString(b.Signature), hex.EncodeToString(b.PubKey[:]))
}

// BroadcastMarkSendRecord mark send message record to prevent messages from being sent repeatedly.
func (b *BlockVerificationMsg) BroadcastMarkSendRecord(ps *peers.PeerSet, peers []string) {
	for _, peer := range peers {
		ps.MarkBlockVerification(peer, b.Signature)
	}
}

// BroadcastFilterTargetPeers filter target peers to filter the nodes that need to send messages.
func (b *BlockVerificationMsg) BroadcastFilterTargetPeers(ps *peers.PeerSet) []string {
	return ps.PeersWithoutSignature(b.Signature)
}

// BlockProposeMsg block propose message transferred between nodes.
type BlockProposeMsg struct {
	RawBlock []byte
}

// NewBlockProposeMsg create new block propose msg.
func NewBlockProposeMsg(block *types.Block) (ConsensusMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockProposeMsg{RawBlock: rawBlock}, nil
}

// GetProposeBlock get propose block from msg.
func (bp *BlockProposeMsg) GetProposeBlock() (*types.Block, error) {
	block := &types.Block{}
	if err := block.UnmarshalText(bp.RawBlock); err != nil {
		return nil, err
	}
	return block, nil
}

func (bp *BlockProposeMsg) String() string {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return "{err: wrong message}"
	}
	blockHash := block.Hash()
	return fmt.Sprintf("{block_height: %d, block_hash: %s}", block.Height, blockHash.String())
}

// BroadcastMarkSendRecord mark send message record to prevent messages from being sent repeatedly.
func (bp *BlockProposeMsg) BroadcastMarkSendRecord(ps *peers.PeerSet, peers []string) {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return
	}

	hash := block.Hash()
	height := block.Height
	for _, peer := range peers {
		ps.MarkBlock(peer, &hash)
		ps.MarkStatus(peer, height)
	}
}

// BroadcastFilterTargetPeers filter target peers to filter the nodes that need to send messages.
func (bp *BlockProposeMsg) BroadcastFilterTargetPeers(ps *peers.PeerSet) []string {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return nil
	}

	return ps.PeersWithoutBlock(block.Hash())
}
