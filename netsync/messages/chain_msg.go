package messages

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
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
	StatusByte          = byte(0x21)
	NewTransactionByte  = byte(0x30)
	NewTransactionsByte = byte(0x31)
	NewMineBlockByte    = byte(0x40)
	FilterLoadByte      = byte(0x50)
	FilterAddByte       = byte(0x51)
	FilterClearByte     = byte(0x52)
	MerkleRequestByte   = byte(0x60)
	MerkleResponseByte  = byte(0x61)

	MaxBlockchainResponseSize = 22020096 + 2
	TxsMsgMaxTxNum            = 1024
)

//BlockchainMessage is a generic message for this reactor.
type BlockchainMessage interface {
	String() string
}

var _ = wire.RegisterInterface(
	struct{ BlockchainMessage }{},
	wire.ConcreteType{&GetBlockMessage{}, BlockRequestByte},
	wire.ConcreteType{&BlockMessage{}, BlockResponseByte},
	wire.ConcreteType{&GetHeadersMessage{}, HeadersRequestByte},
	wire.ConcreteType{&HeadersMessage{}, HeadersResponseByte},
	wire.ConcreteType{&GetBlocksMessage{}, BlocksRequestByte},
	wire.ConcreteType{&BlocksMessage{}, BlocksResponseByte},
	wire.ConcreteType{&StatusMessage{}, StatusByte},
	wire.ConcreteType{&TransactionMessage{}, NewTransactionByte},
	wire.ConcreteType{&TransactionsMessage{}, NewTransactionsByte},
	wire.ConcreteType{&MineBlockMessage{}, NewMineBlockByte},
	wire.ConcreteType{&FilterLoadMessage{}, FilterLoadByte},
	wire.ConcreteType{&FilterAddMessage{}, FilterAddByte},
	wire.ConcreteType{&FilterClearMessage{}, FilterClearByte},
	wire.ConcreteType{&GetMerkleBlockMessage{}, MerkleRequestByte},
	wire.ConcreteType{&MerkleBlockMessage{}, MerkleResponseByte},
)

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

func (m *GetBlockMessage) String() string {
	if m.Height > 0 {
		return fmt.Sprintf("{height: %d}", m.Height)
	}
	return fmt.Sprintf("{hash: %s}", hex.EncodeToString(m.RawHash[:]))
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
func (m *BlockMessage) GetBlock() (*types.Block, error) {
	block := &types.Block{
		BlockHeader:  types.BlockHeader{},
		Transactions: []*types.Tx{},
	}
	if err := block.UnmarshalText(m.RawBlock); err != nil {
		return nil, err
	}
	return block, nil
}

func (m *BlockMessage) String() string {
	block, err := m.GetBlock()
	if err != nil {
		return "{err: wrong message}"
	}
	blockHash := block.Hash()
	return fmt.Sprintf("{block_height: %d, block_hash: %s}", block.Height, blockHash.String())
}

//GetHeadersMessage is one of the bytom msg type
type GetHeadersMessage struct {
	RawBlockLocator [][32]byte
	RawStopHash     [32]byte
	Skip            uint64
}

//NewGetHeadersMessage return a new GetHeadersMessage
func NewGetHeadersMessage(blockLocator []*bc.Hash, stopHash *bc.Hash, skip uint64) *GetHeadersMessage {
	msg := &GetHeadersMessage{
		RawStopHash: stopHash.Byte32(),
		Skip:        skip,
	}
	for _, hash := range blockLocator {
		msg.RawBlockLocator = append(msg.RawBlockLocator, hash.Byte32())
	}
	return msg
}

//GetBlockLocator return the locator of the msg
func (m *GetHeadersMessage) GetBlockLocator() []*bc.Hash {
	blockLocator := []*bc.Hash{}
	for _, rawHash := range m.RawBlockLocator {
		hash := bc.NewHash(rawHash)
		blockLocator = append(blockLocator, &hash)
	}
	return blockLocator
}

func (m *GetHeadersMessage) String() string {
	stopHash := bc.NewHash(m.RawStopHash)
	return fmt.Sprintf("{skip:%d,stopHash:%s}", m.Skip, stopHash.String())
}

//GetStopHash return the stop hash of the msg
func (m *GetHeadersMessage) GetStopHash() *bc.Hash {
	hash := bc.NewHash(m.RawStopHash)
	return &hash
}

func (m *GetHeadersMessage) GetSkip() uint64 {
	return m.Skip
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
func (m *HeadersMessage) GetHeaders() ([]*types.BlockHeader, error) {
	headers := []*types.BlockHeader{}
	for _, data := range m.RawHeaders {
		header := &types.BlockHeader{}
		if err := json.Unmarshal(data, header); err != nil {
			return nil, err
		}

		headers = append(headers, header)
	}
	return headers, nil
}

func (m *HeadersMessage) String() string {
	return fmt.Sprintf("{header_length: %d}", len(m.RawHeaders))
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
func (m *GetBlocksMessage) GetBlockLocator() []*bc.Hash {
	blockLocator := []*bc.Hash{}
	for _, rawHash := range m.RawBlockLocator {
		hash := bc.NewHash(rawHash)
		blockLocator = append(blockLocator, &hash)
	}
	return blockLocator
}

//GetStopHash return the stop hash of the msg
func (m *GetBlocksMessage) GetStopHash() *bc.Hash {
	hash := bc.NewHash(m.RawStopHash)
	return &hash
}

func (m *GetBlocksMessage) String() string {
	return fmt.Sprintf("{stop_hash: %s}", hex.EncodeToString(m.RawStopHash[:]))
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
func (m *BlocksMessage) GetBlocks() ([]*types.Block, error) {
	blocks := []*types.Block{}
	for _, data := range m.RawBlocks {
		block := &types.Block{}
		if err := json.Unmarshal(data, block); err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (m *BlocksMessage) String() string {
	return fmt.Sprintf("{blocks_length: %d}", len(m.RawBlocks))
}

//StatusResponseMessage get status response msg
type StatusMessage struct {
	BestHeight      uint64
	BestHash        [32]byte
	JustifiedHeight uint64
	JustifiedHash   [32]byte
}

//NewStatusResponseMessage construct get status response msg
func NewStatusMessage(bestHeader, justifiedHeader *types.BlockHeader) *StatusMessage {
	return &StatusMessage{
		BestHeight:      bestHeader.Height,
		BestHash:        bestHeader.Hash().Byte32(),
		JustifiedHeight: justifiedHeader.Height,
		JustifiedHash:   justifiedHeader.Hash().Byte32(),
	}
}

//GetHash get hash from msg
func (m *StatusMessage) GetBestHash() *bc.Hash {
	hash := bc.NewHash(m.BestHash)
	return &hash
}

func (m *StatusMessage) GetIrreversibleHash() *bc.Hash {
	hash := bc.NewHash(m.JustifiedHash)
	return &hash
}

func (m *StatusMessage) String() string {
	return fmt.Sprintf("{best hash: %s, irreversible hash: %s}", hex.EncodeToString(m.BestHash[:]), hex.EncodeToString(m.JustifiedHash[:]))
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

func (m *TransactionMessage) String() string {
	tx, err := m.GetTransaction()
	if err != nil {
		return "{err: wrong message}"
	}
	return fmt.Sprintf("{tx_size: %d, tx_hash: %s}", len(m.RawTx), tx.ID.String())
}

//TransactionsMessage notify new txs msg
type TransactionsMessage struct {
	RawTxs [][]byte
}

//NewTransactionsMessage construct notify new txs msg
func NewTransactionsMessage(txs []*types.Tx) (*TransactionsMessage, error) {
	rawTxs := make([][]byte, 0, len(txs))
	for _, tx := range txs {
		rawTx, err := tx.TxData.MarshalText()
		if err != nil {
			return nil, err
		}

		rawTxs = append(rawTxs, rawTx)
	}
	return &TransactionsMessage{RawTxs: rawTxs}, nil
}

//GetTransactions get txs from msg
func (m *TransactionsMessage) GetTransactions() ([]*types.Tx, error) {
	txs := make([]*types.Tx, 0, len(m.RawTxs))
	for _, rawTx := range m.RawTxs {
		tx := &types.Tx{}
		if err := tx.UnmarshalText(rawTx); err != nil {
			return nil, err
		}

		txs = append(txs, tx)
	}
	return txs, nil
}

func (m *TransactionsMessage) String() string {
	return fmt.Sprintf("{tx_num: %d}", len(m.RawTxs))
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

func (m *MineBlockMessage) String() string {
	block, err := m.GetMineBlock()
	if err != nil {
		return "{err: wrong message}"
	}
	blockHash := block.Hash()
	return fmt.Sprintf("{block_height: %d, block_hash: %s}", block.Height, blockHash.String())
}

//FilterLoadMessage tells the receiving peer to filter the transactions according to address.
type FilterLoadMessage struct {
	Addresses [][]byte
}

func (m *FilterLoadMessage) String() string {
	return fmt.Sprintf("{addresses_length: %d}", len(m.Addresses))
}

// FilterAddMessage tells the receiving peer to add address to the filter.
type FilterAddMessage struct {
	Address []byte
}

func (m *FilterAddMessage) String() string {
	return fmt.Sprintf("{address: %s}", hex.EncodeToString(m.Address))
}

//FilterClearMessage tells the receiving peer to remove a previously-set filter.
type FilterClearMessage struct{}

func (m *FilterClearMessage) String() string {
	return "{}"
}

//GetMerkleBlockMessage request merkle blocks from remote peers by height/hash
type GetMerkleBlockMessage struct {
	Height  uint64
	RawHash [32]byte
}

//GetHash reutrn the hash of the request
func (m *GetMerkleBlockMessage) GetHash() *bc.Hash {
	hash := bc.NewHash(m.RawHash)
	return &hash
}

func (m *GetMerkleBlockMessage) String() string {
	if m.Height > 0 {
		return fmt.Sprintf("{height: %d}", m.Height)
	}
	return fmt.Sprintf("{hash: %s}", hex.EncodeToString(m.RawHash[:]))
}

//MerkleBlockMessage return the merkle block to client
type MerkleBlockMessage struct {
	RawBlockHeader []byte
	TxHashes       [][32]byte
	RawTxDatas     [][]byte
	Flags          []byte
}

func (m *MerkleBlockMessage) SetRawBlockHeader(bh types.BlockHeader) error {
	rawHeader, err := bh.MarshalText()
	if err != nil {
		return err
	}

	m.RawBlockHeader = rawHeader
	return nil
}

func (m *MerkleBlockMessage) SetTxInfo(txHashes []*bc.Hash, txFlags []uint8, relatedTxs []*types.Tx) error {
	for _, txHash := range txHashes {
		m.TxHashes = append(m.TxHashes, txHash.Byte32())
	}
	for _, tx := range relatedTxs {
		rawTxData, err := tx.MarshalText()
		if err != nil {
			return err
		}

		m.RawTxDatas = append(m.RawTxDatas, rawTxData)
	}
	m.Flags = txFlags
	return nil
}

func (m *MerkleBlockMessage) String() string {
	return "{}"
}

//NewMerkleBlockMessage construct merkle block message
func NewMerkleBlockMessage() *MerkleBlockMessage {
	return &MerkleBlockMessage{}
}
