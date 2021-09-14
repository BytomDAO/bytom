package integration

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/bytom/bytom/database"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/testutil"
)

const (
	dbDir = "temp"
)

type storeItem struct {
	key []byte
	val interface{}
}

type storeItems []*storeItem

type processBlockTestCase struct {
	desc             string
	initStore        []storeEntry
	wantStore        []storeEntry
	initOrphanManage *protocol.OrphanManage
	wantOrphanManage *protocol.OrphanManage
	wantIsOrphan     bool
	wantError        bool
	newBlock         *types.Block
}

func (p *processBlockTestCase) Run() error {
	defer os.RemoveAll(dbDir)
	if p.initStore == nil {
		p.initStore = make([]storeEntry, 0)
	}
	store, db, err := initStore(p)
	if err != nil {
		return err
	}

	orphanManage := p.initOrphanManage
	if orphanManage == nil {
		orphanManage = protocol.NewOrphanManage()
	}

	txPool := protocol.NewTxPool(store, event.NewDispatcher())
	chain, err := protocol.NewChainWithOrphanManage(store, txPool, orphanManage, nil)
	if err != nil {
		return err
	}

	isOrphan, err := chain.ProcessBlock(p.newBlock)
	if p.wantError != (err != nil) {
		return fmt.Errorf("#case(%s) want error:%t, got error:%t", p.desc, p.wantError, err != nil)
	}

	if isOrphan != p.wantIsOrphan {
		return fmt.Errorf("#case(%s) want orphan:%t, got orphan:%t", p.desc, p.wantIsOrphan, isOrphan)
	}

	if p.wantStore != nil {
		gotStoreEntries := loadStoreEntries(db)
		if !equalsStoreEntries(p.wantStore, gotStoreEntries) {
			gotMap := make(map[string]string)
			for _, entry := range gotStoreEntries {
				gotMap[hex.EncodeToString(entry.key)] = hex.EncodeToString(entry.val)
			}

			wantMap := make(map[string]string)
			for _, entry := range p.wantStore {
				wantMap[hex.EncodeToString(entry.key)] = hex.EncodeToString(entry.val)
			}
			return fmt.Errorf("#case(%s) want store:%v, got store:%v", p.desc, p.wantStore, gotStoreEntries)
		}
	}

	if p.wantOrphanManage != nil {
		if !orphanManage.Equals(p.wantOrphanManage) {
			return fmt.Errorf("#case(%s) want orphan manage:%v, got orphan manage:%v", p.desc, *p.wantOrphanManage, *orphanManage)
		}
	}
	return nil
}

func initStore(c *processBlockTestCase) (state.Store, dbm.DB, error) {
	testDB := dbm.NewDB("testdb", "leveldb", dbDir)
	batch := testDB.NewBatch()
	for _, entry := range c.initStore {
		batch.Set(entry.key, entry.val)
	}
	batch.Write()
	return database.NewStore(testDB), testDB, nil
}

func sortSpendOutputID(block *types.Block) {
	for _, tx := range block.Transactions {
		sort.Sort(HashSlice(tx.SpentOutputIDs))
	}
}

type HashSlice []bc.Hash

func (p HashSlice) Len() int           { return len(p) }
func (p HashSlice) Less(i, j int) bool { return p[i].String() < p[j].String() }
func (p HashSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type storeEntry struct {
	key []byte
	val []byte
}

func serialItem(item *storeItem) ([]storeEntry, error) {
	var storeEntrys []storeEntry
	switch item.val.(type) {
	case *state.BlockStoreState:
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
