package database

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	colon = byte(0x3a)

	blockHashes byte = iota + 1
	blockHeader
	blockTransactions
	mainChainIndex
	checkpoint
	utxo
	contract
)

var (
	// BlockHashesKeyPrefix key Prefix
	BlockHashesKeyPrefix    = []byte{blockHashes, colon}
	blockHeaderKeyPrefix    = []byte{blockHeader, colon}
	blockTransactionsKey    = []byte{blockTransactions, colon}
	mainChainIndexKeyPrefix = []byte{mainChainIndex, colon}
	checkpointKeyPrefix     = []byte{checkpoint, colon}
	UtxoKeyPrefix           = []byte{utxo, colon}
	ContractPrefix          = []byte{contract, colon}
)

func calcMainChainIndexPrefix(height uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], height)
	return append(mainChainIndexKeyPrefix, buf[:]...)
}

// CalcBlockHeaderKey make up header key with prefix + hash
func CalcBlockHeaderKey(hash *bc.Hash) []byte {
	return append(blockHeaderKeyPrefix, hash.Bytes()...)
}

// CalcBlockHashesKey make up hashes key with prefix + height
func CalcBlockHashesKey(height uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], height)
	return append(BlockHashesKeyPrefix, buf[:]...)
}

// CalcBlockTransactionsKey make up txs key with prefix + hash
func CalcBlockTransactionsKey(hash *bc.Hash) []byte {
	return append(blockTransactionsKey, hash.Bytes()...)
}

// GetBlockHeader return the block header by given hash
func GetBlockHeader(db dbm.DB, hash *bc.Hash) (*types.BlockHeader, error) {
	binaryBlockHeader := db.Get(CalcBlockHeaderKey(hash))
	if binaryBlockHeader == nil {
		return nil, fmt.Errorf("There are no blockHeader with given hash %s", hash.String())
	}

	blockHeader := &types.BlockHeader{}
	if err := blockHeader.UnmarshalText(binaryBlockHeader); err != nil {
		return nil, err
	}
	return blockHeader, nil
}

// GetBlockTransactions return the block transactions by given hash
func GetBlockTransactions(db dbm.DB, hash *bc.Hash) ([]*types.Tx, error) {
	binaryBlockTxs := db.Get(CalcBlockTransactionsKey(hash))
	if binaryBlockTxs == nil {
		return nil, fmt.Errorf("There are no block transactions with given hash %s", hash.String())
	}

	block := &types.Block{}
	if err := block.UnmarshalText(binaryBlockTxs); err != nil {
		return nil, err
	}
	return block.Transactions, nil
}

// GetBlockHashesByHeight return block hashes by given height
func GetBlockHashesByHeight(db dbm.DB, height uint64) ([]*bc.Hash, error) {
	binaryHashes := db.Get(CalcBlockHashesKey(height))
	if binaryHashes == nil {
		return []*bc.Hash{}, nil
	}

	hashes := []*bc.Hash{}
	if err := json.Unmarshal(binaryHashes, &hashes); err != nil {
		return nil, err
	}
	return hashes, nil
}

// GetMainChainHash return BlockHash by given height
func GetMainChainHash(db dbm.DB, height uint64) (*bc.Hash, error) {
	binaryHash := db.Get(calcMainChainIndexPrefix(height))
	if binaryHash == nil {
		return nil, fmt.Errorf("There are no BlockHash with given height %d", height)
	}

	hash := &bc.Hash{}
	if err := hash.UnmarshalText(binaryHash); err != nil {
		return nil, err
	}

	return hash, nil
}
