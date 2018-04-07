package netsync

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

//protocol msg
const (
	BlockRequestByte   = byte(0x10)
	BlockResponseByte  = byte(0x11)
	StatusRequestByte  = byte(0x20)
	StatusResponseByte = byte(0x21)
	NewTransactionByte = byte(0x30)
	NewMineBlockByte   = byte(0x40)

	maxBlockchainResponseSize = 22020096 + 2
)

// BlockchainMessage is a generic message for this reactor.
type BlockchainMessage interface{}

var _ = wire.RegisterInterface(
	struct{ BlockchainMessage }{},
	wire.ConcreteType{&BlockRequestMessage{}, BlockRequestByte},
	wire.ConcreteType{&BlockResponseMessage{}, BlockResponseByte},
	wire.ConcreteType{&StatusRequestMessage{}, StatusRequestByte},
	wire.ConcreteType{&StatusResponseMessage{}, StatusResponseByte},
	wire.ConcreteType{&TransactionNotifyMessage{}, NewTransactionByte},
	wire.ConcreteType{&MineBlockMessage{}, NewMineBlockByte},
)

type blockPending struct {
	block  *types.Block
	peerID string
}

type txsNotify struct {
	tx     *types.Tx
	peerID string
}

//DecodeMessage decode msg
func DecodeMessage(bz []byte) (msgType byte, msg BlockchainMessage, err error) {
	msgType = bz[0]
	n := int(0)
	r := bytes.NewReader(bz)
	msg = wire.ReadBinary(struct{ BlockchainMessage }{}, r, maxBlockchainResponseSize, &n, &err).(struct{ BlockchainMessage }).BlockchainMessage
	if err != nil && n != len(bz) {
		err = errors.New("DecodeMessage() had bytes left over")
	}
	return
}

//BlockRequestMessage request blocks from remote peers by height/hash
type BlockRequestMessage struct {
	Height  uint64
	RawHash [32]byte
}

//GetHash get hash
func (m *BlockRequestMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

//String convert msg to string
func (m *BlockRequestMessage) String() string {
	if m.Height > 0 {
		return fmt.Sprintf("BlockRequestMessage{Height: %d}", m.Height)
	}
	hash := m.GetHash()
	return fmt.Sprintf("BlockRequestMessage{Hash: %s}", hash.String())
}

//BlockResponseMessage response get block msg
type BlockResponseMessage struct {
	RawBlock []byte
}

//NewBlockResponseMessage construct bock response msg
func NewBlockResponseMessage(block *types.Block) (*BlockResponseMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockResponseMessage{RawBlock: rawBlock}, nil
}

//GetBlock get block from msg
func (m *BlockResponseMessage) GetBlock() *types.Block {
	block := &types.Block{
		BlockHeader:  types.BlockHeader{},
		Transactions: []*types.Tx{},
	}
	block.UnmarshalText(m.RawBlock)
	return block
}

//String convert msg to string
func (m *BlockResponseMessage) String() string {
	return fmt.Sprintf("BlockResponseMessage{Size: %d}", len(m.RawBlock))
}

//TransactionNotifyMessage notify new tx msg
type TransactionNotifyMessage struct {
	RawTx []byte
}

//NewTransactionNotifyMessage construct notify new tx msg
func NewTransactionNotifyMessage(tx *types.Tx) (*TransactionNotifyMessage, error) {
	rawTx, err := tx.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	return &TransactionNotifyMessage{RawTx: rawTx}, nil
}

//GetTransaction get tx from msg
func (m *TransactionNotifyMessage) GetTransaction() (*types.Tx, error) {
	tx := &types.Tx{}
	if err := tx.UnmarshalText(m.RawTx); err != nil {
		return nil, err
	}
	return tx, nil
}

//String
func (m *TransactionNotifyMessage) String() string {
	return fmt.Sprintf("TransactionNotifyMessage{Size: %d}", len(m.RawTx))
}

//StatusRequestMessage status request msg
type StatusRequestMessage struct{}

//String
func (m *StatusRequestMessage) String() string {
	return "StatusRequestMessage"
}

//StatusResponseMessage get status response msg
type StatusResponseMessage struct {
	Height  uint64
	RawHash [32]byte
}

//NewStatusResponseMessage construct get status response msg
func NewStatusResponseMessage(block *types.Block) *StatusResponseMessage {
	return &StatusResponseMessage{
		Height:  block.Height,
		RawHash: block.Hash().Byte32(),
	}
}

//GetHash get hash from msg
func (m *StatusResponseMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

//String convert msg to string
func (m *StatusResponseMessage) String() string {
	hash := m.GetHash()
	return fmt.Sprintf("StatusResponseMessage{Height: %d, Hash: %s}", m.Height, hash.String())
}

//MineBlockMessage new mined block msg
type MineBlockMessage struct {
	RawBlock []byte
}

//NewMinedBlockMessage construct new mined block msg
func NewMinedBlockMessage(block *types.Block) (*MineBlockMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &MineBlockMessage{RawBlock: rawBlock}, nil
}

//GetMineBlock get mine block from msg
func (m *MineBlockMessage) GetMineBlock() (*types.Block, error) {
	block := &types.Block{}
	if err:=block.UnmarshalText(m.RawBlock);err!=nil{
		return nil, err
	}
	return block, nil
}

//String convert msg to string
func (m *MineBlockMessage) String() string {
	return fmt.Sprintf("NewMineBlockMessage{Size: %d}", len(m.RawBlock))
}
