package blockchain

import (
	"bytes"
	"errors"
	"fmt"

	wire "github.com/tendermint/go-wire"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	//BlockRequestByte means block request message
	BlockRequestByte = byte(0x10)
	//BlockResponseByte means block response message
	BlockResponseByte = byte(0x11)
	//StatusRequestByte means status request message
	StatusRequestByte = byte(0x20)
	//StatusResponseByte means status response message
	StatusResponseByte = byte(0x21)
	//NewTransactionByte means transaction notify message
	NewTransactionByte = byte(0x30)
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
)

//DecodeMessage decode receive messages
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

//BlockRequestMessage is block request message struct
type BlockRequestMessage struct {
	Height  uint64
	RawHash [32]byte
}

//GetHash return block hash
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

//BlockResponseMessage is block response message struct
type BlockResponseMessage struct {
	RawBlock []byte
}

//NewBlockResponseMessage produce new BlockResponseMessage instance
func NewBlockResponseMessage(block *legacy.Block) (*BlockResponseMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockResponseMessage{RawBlock: rawBlock}, nil
}

//GetBlock return block struct
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

//TransactionNotifyMessage is transaction notify message struct
type TransactionNotifyMessage struct {
	RawTx []byte
}

//NewTransactionNotifyMessage produce new TransactionNotifyMessage instance
func NewTransactionNotifyMessage(tx *legacy.Tx) (*TransactionNotifyMessage, error) {
	rawTx, err := tx.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	return &TransactionNotifyMessage{RawTx: rawTx}, nil
}

//GetTransaction return Tx struct
func (m *TransactionNotifyMessage) GetTransaction() *legacy.Tx {
	tx := &legacy.Tx{}
	tx.UnmarshalText(m.RawTx)
	return tx
}

func (m *TransactionNotifyMessage) String() string {
	return fmt.Sprintf("TransactionNotifyMessage{Size: %d}", len(m.RawTx))
}

//StatusRequestMessage is status request message struct
type StatusRequestMessage struct{}

func (m *StatusRequestMessage) String() string {
	return "StatusRequestMessage"
}

//StatusResponseMessage is status response message struct
type StatusResponseMessage struct {
	Height  uint64
	RawHash [32]byte
}

//NewStatusResponseMessage produce new StatusResponseMessage instance
func NewStatusResponseMessage(block *legacy.Block) *StatusResponseMessage {
	return &StatusResponseMessage{
		Height:  block.Height,
		RawHash: block.Hash().Byte32(),
	}
}

//GetHash return hash pointer
func (m *StatusResponseMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

func (m *StatusResponseMessage) String() string {
	hash := m.GetHash()
	return fmt.Sprintf("StatusResponseMessage{Height: %d, Hash: %s}", m.Height, hash.String())
}
