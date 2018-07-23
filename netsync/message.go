package netsync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

//protocol msg byte
const (
	BlockchainChannel = byte(0x40)

	BlockRequestByte    = byte(0x10)
	BlockResponseByte   = byte(0x11)
	HeadersRequestByte  = byte(0x12)
	HeadersResponseByte = byte(0x13)
	BlocksRequestByte   = byte(0x14)
	BlocksResponseByte  = byte(0x15)
	StatusRequestByte   = byte(0x20)
	StatusResponseByte  = byte(0x21)
	NewTransactionByte  = byte(0x30)
	NewMineBlockByte    = byte(0x40)

	maxBlockchainResponseSize = 22020096 + 2
)

//BlockchainMessage is a generic message for this reactor.
type BlockchainMessage interface{}

var _ = wire.RegisterInterface(
	struct{ BlockchainMessage }{},
	wire.ConcreteType{&GetBlockMessage{}, BlockRequestByte},
	wire.ConcreteType{&BlockMessage{}, BlockResponseByte},
	wire.ConcreteType{&GetHeadersMessage{}, HeadersRequestByte},
	wire.ConcreteType{&HeadersMessage{}, HeadersResponseByte},
	wire.ConcreteType{&GetBlocksMessage{}, BlocksRequestByte},
	wire.ConcreteType{&BlocksMessage{}, BlocksResponseByte},
	wire.ConcreteType{&StatusRequestMessage{}, StatusRequestByte},
	wire.ConcreteType{&StatusResponseMessage{}, StatusResponseByte},
	wire.ConcreteType{&TransactionMessage{}, NewTransactionByte},
	wire.ConcreteType{&MineBlockMessage{}, NewMineBlockByte},
)

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

//GetBlockMessage request blocks from remote peers by height/hash
type GetBlockMessage struct {
	Height  uint64
	RawHash [32]byte
}

//GetHash reutrn the hash of the request
func (m *GetBlockMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

//String convert msg to string
func (m *GetBlockMessage) String() string {
	if m.Height > 0 {
		return fmt.Sprintf("GetBlockMessage{Height: %d}", m.Height)
	}
	hash := m.GetHash()
	return fmt.Sprintf("GetBlockMessage{Hash: %s}", hash.String())
}

//BlockMessage response get block msg
type BlockMessage struct {
	RawBlock []byte
}

//NewBlockMessage construct bock response msg
func NewBlockMessage(block *types.Block) (*BlockMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockMessage{RawBlock: rawBlock}, nil
}

//GetBlock get block from msg
func (m *BlockMessage) GetBlock() *types.Block {
	block := &types.Block{
		BlockHeader:  types.BlockHeader{},
		Transactions: []*types.Tx{},
	}
	block.UnmarshalText(m.RawBlock)
	return block
}

//String convert msg to string
func (m *BlockMessage) String() string {
	return fmt.Sprintf("BlockMessage{Size: %d}", len(m.RawBlock))
}

//GetHeadersMessage is one of the bytom msg type
type GetHeadersMessage struct {
	RawBlockLocator [][32]byte
	RawStopHash     [32]byte
}

//NewGetHeadersMessage return a new GetHeadersMessage
func NewGetHeadersMessage(blockLocator []*bc.Hash, stopHash *bc.Hash) *GetHeadersMessage {
	msg := &GetHeadersMessage{
		RawStopHash: stopHash.Byte32(),
	}
	for _, hash := range blockLocator {
		msg.RawBlockLocator = append(msg.RawBlockLocator, hash.Byte32())
	}
	return msg
}

//GetBlockLocator return the locator of the msg
func (msg *GetHeadersMessage) GetBlockLocator() []*bc.Hash {
	blockLocator := []*bc.Hash{}
	for _, rawHash := range msg.RawBlockLocator {
		hash := bc.NewHash(rawHash)
		blockLocator = append(blockLocator, &hash)
	}
	return blockLocator
}

//GetStopHash return the stop hash of the msg
func (msg *GetHeadersMessage) GetStopHash() *bc.Hash {
	hash := bc.NewHash(msg.RawStopHash)
	return &hash
}

//HeadersMessage is one of the bytom msg type
type HeadersMessage struct {
	RawHeaders [][]byte
}

//NewHeadersMessage create a new HeadersMessage
func NewHeadersMessage(headers []*types.BlockHeader) (*HeadersMessage, error) {
	RawHeaders := [][]byte{}
	for _, header := range headers {
		data, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}

		RawHeaders = append(RawHeaders, data)
	}
	return &HeadersMessage{RawHeaders: RawHeaders}, nil
}

//GetHeaders return the headers in the msg
func (msg *HeadersMessage) GetHeaders() ([]*types.BlockHeader, error) {
	headers := []*types.BlockHeader{}
	for _, data := range msg.RawHeaders {
		header := &types.BlockHeader{}
		if err := json.Unmarshal(data, header); err != nil {
			return nil, err
		}

		headers = append(headers, header)
	}
	return headers, nil
}

//GetBlocksMessage is one of the bytom msg type
type GetBlocksMessage struct {
	RawBlockLocator [][32]byte
	RawStopHash     [32]byte
}

//NewGetBlocksMessage create a new GetBlocksMessage
func NewGetBlocksMessage(blockLocator []*bc.Hash, stopHash *bc.Hash) *GetBlocksMessage {
	msg := &GetBlocksMessage{
		RawStopHash: stopHash.Byte32(),
	}
	for _, hash := range blockLocator {
		msg.RawBlockLocator = append(msg.RawBlockLocator, hash.Byte32())
	}
	return msg
}

//GetBlockLocator return the locator of the msg
func (msg *GetBlocksMessage) GetBlockLocator() []*bc.Hash {
	blockLocator := []*bc.Hash{}
	for _, rawHash := range msg.RawBlockLocator {
		hash := bc.NewHash(rawHash)
		blockLocator = append(blockLocator, &hash)
	}
	return blockLocator
}

//GetStopHash return the stop hash of the msg
func (msg *GetBlocksMessage) GetStopHash() *bc.Hash {
	hash := bc.NewHash(msg.RawStopHash)
	return &hash
}

//BlocksMessage is one of the bytom msg type
type BlocksMessage struct {
	RawBlocks [][]byte
}

//NewBlocksMessage create a new BlocksMessage
func NewBlocksMessage(blocks []*types.Block) (*BlocksMessage, error) {
	rawBlocks := [][]byte{}
	for _, block := range blocks {
		data, err := json.Marshal(block)
		if err != nil {
			return nil, err
		}

		rawBlocks = append(rawBlocks, data)
	}
	return &BlocksMessage{RawBlocks: rawBlocks}, nil
}

//GetBlocks returns the blocks in the msg
func (msg *BlocksMessage) GetBlocks() ([]*types.Block, error) {
	blocks := []*types.Block{}
	for _, data := range msg.RawBlocks {
		block := &types.Block{}
		if err := json.Unmarshal(data, block); err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

//StatusRequestMessage status request msg
type StatusRequestMessage struct{}

//String
func (m *StatusRequestMessage) String() string {
	return "StatusRequestMessage"
}

//StatusResponseMessage get status response msg
type StatusResponseMessage struct {
	Height      uint64
	RawHash     [32]byte
	GenesisHash [32]byte
}

//NewStatusResponseMessage construct get status response msg
func NewStatusResponseMessage(blockHeader *types.BlockHeader, hash *bc.Hash) *StatusResponseMessage {
	return &StatusResponseMessage{
		Height:      blockHeader.Height,
		RawHash:     blockHeader.Hash().Byte32(),
		GenesisHash: hash.Byte32(),
	}
}

//GetHash get hash from msg
func (m *StatusResponseMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

//GetGenesisHash get hash from msg
func (m *StatusResponseMessage) GetGenesisHash() *bc.Hash {
	hash := bc.NewHash(m.GenesisHash)
	return &hash
}

//String convert msg to string
func (m *StatusResponseMessage) String() string {
	hash := m.GetHash()
	genesisHash := m.GetGenesisHash()
	return fmt.Sprintf("StatusResponseMessage{Height: %d, Best hash: %s, Genesis hash: %s}", m.Height, hash.String(), genesisHash.String())
}

//TransactionMessage notify new tx msg
type TransactionMessage struct {
	RawTx []byte
}

//NewTransactionMessage construct notify new tx msg
func NewTransactionMessage(tx *types.Tx) (*TransactionMessage, error) {
	rawTx, err := tx.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	return &TransactionMessage{RawTx: rawTx}, nil
}

//GetTransaction get tx from msg
func (m *TransactionMessage) GetTransaction() (*types.Tx, error) {
	tx := &types.Tx{}
	if err := tx.UnmarshalText(m.RawTx); err != nil {
		return nil, err
	}
	return tx, nil
}

//String
func (m *TransactionMessage) String() string {
	return fmt.Sprintf("TransactionMessage{Size: %d}", len(m.RawTx))
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
	if err := block.UnmarshalText(m.RawBlock); err != nil {
		return nil, err
	}
	return block, nil
}

//String convert msg to string
func (m *MineBlockMessage) String() string {
	return fmt.Sprintf("NewMineBlockMessage{Size: %d}", len(m.RawBlock))
}
