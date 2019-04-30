package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sort"

	"github.com/golang/protobuf/proto"

	"github.com/bytom/database"
	dbm "github.com/bytom/database/leveldb"
	"github.com/bytom/database/storage"
	"github.com/bytom/event"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	"github.com/bytom/testutil"
)

const (
	dbDir = "temp"
)

type storeItem struct {
	key []byte
	val interface{}
}

type serialFun func(obj interface{}) ([]byte, error)
type deserialFun func(data []byte) (interface{}, error)

func getSerialFun(item interface{}) (serialFun, error) {
	switch item.(type) {
	case *protocol.BlockStoreState:
		return json.Marshal, nil
	case *types.Block:
		return func(obj interface{}) ([]byte, error) {
			block := obj.(*types.Block)
			return block.MarshalText()
		}, nil
	case types.BlockHeader:
		return func(obj interface{}) ([]byte, error) {
			bh := obj.(types.BlockHeader)
			return bh.MarshalText()
		}, nil
	case *bc.TransactionStatus:
		return func(obj interface{}) ([]byte, error) {
			status := obj.(*bc.TransactionStatus)
			return proto.Marshal(status)
		}, nil
	case *storage.UtxoEntry:
		return func(obj interface{}) ([]byte, error) {
			utxo := obj.(*storage.UtxoEntry)
			return proto.Marshal(utxo)
		}, nil
	}
	typ := reflect.TypeOf(item)
	return nil, fmt.Errorf("can not found any serialization function for type:%s", typ.Name())
}

func getDeserialFun(key []byte) (deserialFun, error) {
	funMap := map[string]deserialFun{
		string(database.BlockStoreKey): func(data []byte) (interface{}, error) {
			storeState := &protocol.BlockStoreState{}
			err := json.Unmarshal(data, storeState)
			return storeState, err
		},
		string(database.TxStatusPrefix): func(data []byte) (interface{}, error) {
			status := &bc.TransactionStatus{}
			err := proto.Unmarshal(data, status)
			return status, err
		},
		string(database.BlockPrefix): func(data []byte) (interface{}, error) {
			block := &types.Block{}
			err := block.UnmarshalText(data)
			sortSpendOutputID(block)
			return block, err
		},
		string(database.BlockHeaderPrefix): func(data []byte) (interface{}, error) {
			bh := types.BlockHeader{}
			err := bh.UnmarshalText(data)
			return bh, err
		},
		database.UtxoPreFix: func(data []byte) (interface{}, error) {
			utxo := &storage.UtxoEntry{}
			err := proto.Unmarshal(data, utxo)
			return utxo, err
		},
	}

	for prefix, converter := range funMap {
		if strings.HasPrefix(string(key), prefix) {
			return converter, nil
		}
	}
	return nil, fmt.Errorf("can not found any deserialization function for key:%s", string(key))
}

type storeItems []*storeItem

func (s1 storeItems) equals(s2 storeItems) bool {
	if s2 == nil {
		return false
	}

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

type processBlockTestCase struct {
	desc             string
	initStore        []*storeItem
	wantStore        []*storeItem
	wantBlockIndex   *state.BlockIndex
	initOrphanManage *protocol.OrphanManage
	wantOrphanManage *protocol.OrphanManage
	wantIsOrphan     bool
	wantError        bool
	newBlock         *types.Block
}

func (p *processBlockTestCase) Run() error {
	defer os.RemoveAll(dbDir)
	if p.initStore == nil {
		p.initStore = make([]*storeItem, 0)
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
	chain, err := protocol.NewChainWithOrphanManage(store, txPool, orphanManage)
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
		gotStoreItems, err := loadStoreItems(db)
		if err != nil {
			return err
		}

		if !storeItems(gotStoreItems).equals(p.wantStore) {
			return fmt.Errorf("#case(%s) want store:%v, got store:%v", p.desc, p.wantStore, gotStoreItems)
		}
	}

	if p.wantBlockIndex != nil {
		blockIndex := chain.GetBlockIndex()
		if !blockIndex.Equals(p.wantBlockIndex) {
			return fmt.Errorf("#case(%s) want block index:%v, got block index:%v", p.desc, *p.wantBlockIndex, *blockIndex)
		}
	}

	if p.wantOrphanManage != nil {
		if !orphanManage.Equals(p.wantOrphanManage) {
			return fmt.Errorf("#case(%s) want orphan manage:%v, got orphan manage:%v", p.desc, *p.wantOrphanManage, *orphanManage)
		}
	}
	return nil
}

func loadStoreItems(db dbm.DB) ([]*storeItem, error) {
	iter := db.Iterator()
	defer iter.Release()

	var items []*storeItem
	for iter.Next() {
		item := &storeItem{key: iter.Key()}
		fun, err := getDeserialFun(iter.Key())
		if err != nil {
			return nil, err
		}

		val, err := fun(iter.Value())
		if err != nil {
			return nil, err
		}

		item.val = val
		items = append(items, item)
	}
	return items, nil
}

func initStore(c *processBlockTestCase) (protocol.Store, dbm.DB, error) {
	testDB := dbm.NewDB("testdb", "leveldb", dbDir)
	batch := testDB.NewBatch()
	for _, item := range c.initStore {
		fun, err := getSerialFun(item.val)
		if err != nil {
			return nil, nil, err
		}

		bytes, err := fun(item.val)
		if err != nil {
			return nil, nil, err
		}

		batch.Set(item.key, bytes)
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
