package netsync

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	BlockRequestByte   = byte(0x10)
	BlockResponseByte  = byte(0x11)
	StatusRequestByte  = byte(0x20)
	StatusResponseByte = byte(0x21)
	NewTransactionByte = byte(0x30)
	NewMineBlockByte   = byte(0x40)
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

func (m *BlockRequestMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

func (m *BlockRequestMessage) String() string {
	if m.Height > 0 {
		return fmt.Sprintf("BlockRequestMessage{Height: %d}", m.Height)
	}
	hash := m.GetHash()
	return fmt.Sprintf("BlockRequestMessage{Hash: %s}", hash.String())
}

type BlockResponseMessage struct {
	RawBlock []byte
}

func NewBlockResponseMessage(block *legacy.Block) (*BlockResponseMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockResponseMessage{RawBlock: rawBlock}, nil
}

func (m *BlockResponseMessage) GetBlock() *legacy.Block {
	block := &legacy.Block{
		BlockHeader:  legacy.BlockHeader{},
		Transactions: []*legacy.Tx{},
	}
	block.UnmarshalText(m.RawBlock)
	return block
}

func (m *BlockResponseMessage) String() string {
	return fmt.Sprintf("BlockResponseMessage{Size: %d}", len(m.RawBlock))
}

type TransactionNotifyMessage struct {
	RawTx []byte
}

func NewTransactionNotifyMessage(tx *legacy.Tx) (*TransactionNotifyMessage, error) {
	rawTx, err := tx.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	return &TransactionNotifyMessage{RawTx: rawTx}, nil
}

func (m *TransactionNotifyMessage) GetTransaction() (*legacy.Tx, error) {
	tx := &legacy.Tx{}
	if err := tx.UnmarshalText(m.RawTx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (m *TransactionNotifyMessage) String() string {
	return fmt.Sprintf("TransactionNotifyMessage{Size: %d}", len(m.RawTx))
}

type StatusRequestMessage struct{}

func (m *StatusRequestMessage) String() string {
	return "StatusRequestMessage"
}

type StatusResponseMessage struct {
	Height  uint64
	RawHash [32]byte
}

func NewStatusResponseMessage(block *legacy.Block) *StatusResponseMessage {
	return &StatusResponseMessage{
		Height:  block.Height,
		RawHash: block.Hash().Byte32(),
	}
}

func (m *StatusResponseMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

func (m *StatusResponseMessage) String() string {
	hash := m.GetHash()
	return fmt.Sprintf("StatusResponseMessage{Height: %d, Hash: %s}", m.Height, hash.String())
}

type MineBlockMessage struct {
	RawBlock []byte
}

func NewMinedBlockMessage(block *legacy.Block) (*MineBlockMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &MineBlockMessage{RawBlock: rawBlock}, nil
}

func (m *MineBlockMessage) GetMineBlock() (*legacy.Block, error) {
	block := &legacy.Block{}
	if err:=block.UnmarshalText(m.RawBlock);err!=nil{
		return nil, err
	}
	return block, nil
}

func (m *MineBlockMessage) String() string {
	return fmt.Sprintf("NewMineBlockMessage{Size: %d}", len(m.RawBlock))
}
