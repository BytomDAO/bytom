package integration

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/testutil"
	"github.com/golang/protobuf/proto"
)

type storeEntry struct {
	key []byte
	val []byte
}

func SerialItem(item *storeItem) ([]storeEntry, error) {
	var storeEntrys []storeEntry
	switch item.val.(type) {
	case *protocol.BlockStoreState:
		bytes, err := json.Marshal(item.val)
		if err != nil {
			return nil, err
		}

		storeEntrys = append(storeEntrys, storeEntry{key: item.key, val: bytes})
	case *types.Block:
		block := item.val.(*types.Block)
		hash := block.Hash()
		binaryBlockHeader, err := block.MarshalTextForBlockHeader()
		if err != nil {
			return nil, err
		}

		storeEntrys = append(storeEntrys, storeEntry{key: database.CalcBlockHeaderKey(&hash), val: binaryBlockHeader})
		binaryBlockTxs, err := block.MarshalTextForTransactions()
		if err != nil {
			return nil, err
		}

		storeEntrys = append(storeEntrys, storeEntry{key: database.CalcBlockTransactionsKey(&hash), val: binaryBlockTxs})
	case types.BlockHeader:
		bh := item.val.(types.BlockHeader)
		bytes, err := bh.MarshalText()
		if err != nil {
			return nil, err
		}

		storeEntrys = append(storeEntrys, storeEntry{key: item.key, val: bytes})
	case *storage.UtxoEntry:
		utxo := item.val.(*storage.UtxoEntry)
		bytes, err := proto.Marshal(utxo)
		if err != nil {
			return nil, err
		}

		storeEntrys = append(storeEntrys, storeEntry{key: item.key, val: bytes})
	default:
		typ := reflect.TypeOf(item.val)
		return nil, fmt.Errorf("can not found any serialization function for type:%s", typ.Name())
	}

	return storeEntrys, nil
}

func equalsStoreEntries(s1, s2 []storeEntry) bool {
	itemMap1 := make(map[string]interface{}, len(s1))
	for _, item := range s1 {
		itemMap1[string(item.key)] = item.val
	}

	itemMap2 := make(map[string]interface{}, len(s2))
	for _, item := range s2 {
		itemMap2[string(item.key)] = item.val
	}

	return testutil.DeepEqual(itemMap1, itemMap2)
}

func loadStoreEntries(db dbm.DB) []storeEntry {
	var entries []storeEntry
	iter := db.Iterator()
	defer iter.Release()
	for iter.Next() {
		if strings.HasPrefix(string(iter.Key()), string(database.BlockHashesKeyPrefix)) {
			continue
		}

		item := storeEntry{key: iter.Key(), val: iter.Value()}
		entries = append(entries, item)
	}
	return entries
}
