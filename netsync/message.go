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

//protocol msg
const (
	BlockchainChannel = byte(0x40)

	GetBlockByte        = byte(0x10)
	BlockByte           = byte(0x11)
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

// BlockchainMessage is a generic message for this reactor.
type BlockchainMessage interface{}

var _ = wire.RegisterInterface(
	struct{ BlockchainMessage }{},
	wire.ConcreteType{&GetBlockMessage{}, GetBlockByte},
	wire.ConcreteType{&BlockMessage{}, BlockByte},
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

//GetHash get hash
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

// MsgGetHeaders implements the Message interface and represents a
// getheaders message.  It is used to request a list of block headers for
// blocks starting after the last known hash in the slice of block locator
// hashes.  The list is returned via a headers message (MsgHeaders) and is
// limited by a specific hash to stop at or the maximum number of block headers
// per message, which is currently 2000.
//
// Set the HashStop field to the hash at which to stop and use
// AddBlockLocatorHash to build up the list of block locator hashes.
//
// The algorithm for building the block locator hashes should be to add the
// hashes in reverse order until you reach the genesis block.  In order to keep
// the list of locator hashes to a resonable number of entries, first add the
// most recent 10 block hashes, then double the step each loop iteration to
// exponentially decrease the number of hashes the further away from head and
// closer to the genesis block you get.
type GetHeadersMessage struct {
	RawBlockLocator [][32]byte
	RawStopHash     [32]byte
}

func NewGetHeadersMessage(blockLocator []*bc.Hash, stopHash *bc.Hash) *GetHeadersMessage {
	msg := &GetHeadersMessage{
		RawStopHash: stopHash.Byte32(),
	}
	for _, hash := range blockLocator {
		msg.RawBlockLocator = append(msg.RawBlockLocator, hash.Byte32())
	}
	return msg
}

func (msg *GetHeadersMessage) GetBlockLocator() []*bc.Hash {
	blockLocator := []*bc.Hash{}
	for _, rawHash := range msg.RawBlockLocator {
		hash := bc.NewHash(rawHash)
		blockLocator = append(blockLocator, &hash)
	}
	return blockLocator
}

func (msg *GetHeadersMessage) GetStopHash() *bc.Hash {
	hash := bc.NewHash(msg.RawStopHash)
	return &hash
}

type HeadersMessage struct {
	rawHeaders []byte
}

func NewHeadersMessage(headers []*types.BlockHeader) (*HeadersMessage, error) {
	data, err := json.Marshal(headers)
	if err != nil {
		return nil, err
	}
	return &HeadersMessage{rawHeaders: data}, nil
}

func (msg *HeadersMessage) GetHeaders() ([]*types.BlockHeader, error) {
	headers := []*types.BlockHeader{}
	return headers, json.Unmarshal(msg.rawHeaders, headers)
}

type GetBlocksMessage struct {
	RawBlockLocator [][32]byte
	RawStopHash     [32]byte
}

func NewGetBlocksMessage(blockLocator []*bc.Hash, stopHash *bc.Hash) *GetBlocksMessage {
	msg := &GetBlocksMessage{
		RawStopHash: stopHash.Byte32(),
	}
	for _, hash := range blockLocator {
		msg.RawBlockLocator = append(msg.RawBlockLocator, hash.Byte32())
	}
	return msg
}

func (msg *GetBlocksMessage) GetBlockLocator() []*bc.Hash {
	blockLocator := []*bc.Hash{}
	for _, rawHash := range msg.RawBlockLocator {
		hash := bc.NewHash(rawHash)
		blockLocator = append(blockLocator, &hash)
	}
	return blockLocator
}

func (msg *GetBlocksMessage) GetStopHash() *bc.Hash {
	hash := bc.NewHash(msg.RawStopHash)
	return &hash
}

type BlocksMessage struct {
	RawBlocks []byte
}

func NewBlocksMessage(blocks []*types.Block) (*BlocksMessage, error) {
	data, err := json.Marshal(blocks)
	if err != nil {
		return nil, err
	}
	return &BlocksMessage{RawBlocks: data}, nil
}

func (msg *BlocksMessage) GetBlocks() ([]*types.Block, error) {
	blocks := []*types.Block{}
	return blocks, json.Unmarshal(msg.RawBlocks, blocks)
}
