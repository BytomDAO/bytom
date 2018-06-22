package netsync

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/bytom/common"
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
	GetHeadersByte     = byte(0x50)
	HeadersByte        = byte(0x51)

	maxBlockchainResponseSize = 22020096 + 2

	// MaxBlockLocatorsPerMsg is the maximum number of block locator hashes allowed
	// per message.
	MaxBlockLocatorsPerMsg = 500
	// MaxBlockHeadersPerMsg is the maximum number of block headers that can be in
	// a single bitcoin headers message.
	MaxBlockHeadersPerMsg = 2000
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
	wire.ConcreteType{&GetHeadersMessage{}, GetHeadersByte},
	wire.ConcreteType{&HeadersMessage{}, HeadersByte},
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
	BlockLocatorHashes []*common.Hash
	HashStop           common.Hash
}

// NewMsgGetHeaders returns a new bitcoin getheaders message that conforms to
// the Message interface.  See MsgGetHeaders for details.
func NewMsgGetHeaders() *GetHeadersMessage {
	return &GetHeadersMessage{
		BlockLocatorHashes: make([]*common.Hash, 0, MaxBlockLocatorsPerMsg),
	}
}

// AddBlockLocatorHash adds a new block locator hash to the message.
func (msg *GetHeadersMessage) AddBlockLocatorHash(hash *common.Hash) error {
	if len(msg.BlockLocatorHashes)+1 > MaxBlockLocatorsPerMsg {
		return errors.New("AddBlockLocatorHash too many block locator hashes")
	}
	msg.BlockLocatorHashes = append(msg.BlockLocatorHashes, hash)
	return nil
}

// MsgHeaders implements the Message interface and represents a bitcoin headers
// message.  It is used to deliver block header information in response
// to a getheaders message (MsgGetHeaders).  The maximum number of block headers
// per message is currently 2000.  See MsgGetHeaders for details on requesting
// the headers.
type HeadersMessage struct {
	Headers []*types.BlockHeader
}

//NewTransactionNotifyMessage construct notify new tx msg
func NewHeadersMessage(bh []*types.BlockHeader) (*HeadersMessage, error) {
	return &HeadersMessage{Headers: bh}, nil
}
