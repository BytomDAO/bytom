package contract

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc"
)

const (
	colon = byte(0x3a)

	instance byte = iota + 1
	chainStatus
)

var (
	instancePrefixKey    = []byte{instance, colon}
	chainStatusPrefixKey = []byte{chainStatus, colon}
)

func instanceKey(traceID string) []byte {
	return append(instancePrefixKey, []byte(traceID)...)
}

func chainStatusKey() []byte {
	return chainStatusPrefixKey
}


type TraceStore struct {
	db dbm.DB
}

func NewTraceStore(db dbm.DB) *TraceStore {
	return &TraceStore{db: db}
}

// GetInstance return instance by given trace id
func (t *TraceStore) GetInstance(traceID string) (*Instance, error) {
	key := instanceKey(traceID)
	data := t.db.Get(key)
	instance := &Instance{}
	if err := json.Unmarshal(data, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// LoadInstances used to load all instances in db
func (t *TraceStore) LoadInstances() ([]*Instance, error) {
	iter := t.db.IteratorPrefix(instancePrefixKey)
	defer iter.Release()

	var instances []*Instance
	for iter.Next() {
		instance := &Instance{}
		if err := json.Unmarshal(iter.Value(), instance); err != nil {
			return nil, err
		}

		instances = append(instances, instance)
	}
	return instances, nil
}

// SaveInstances used to batch save multiple instances
func (t *TraceStore) SaveInstances(instances []*Instance) error {
	batch := t.db.NewBatch()
	if err := t.saveInstances(instances, batch); err != nil {
		return err
	}

	batch.Write()
	return nil
}

// RemoveInstance delete a instance by given trace id
func (t *TraceStore) RemoveInstance(traceID string) {
	key := instanceKey(traceID)
	t.db.Delete(key)
}

// SaveInstancesWithStatus batch save the instances and chain status
func (t *TraceStore) SaveInstancesWithStatus(instances []*Instance, blockHeight uint64, blockHash bc.Hash) error {
	batch := t.db.NewBatch()
	if err := t.saveInstances(instances, batch); err != nil {
		return err
	}

	chainData, err := json.Marshal(&ChainStatus{BlockHeight: blockHeight, BlockHash: blockHash})
	if err != nil {
		return err
	}

	batch.Set(chainStatusKey(), chainData)
	batch.Write()
	return nil
}

// GetChainStatus return the current chain status
func (t *TraceStore) GetChainStatus() *ChainStatus {
	data := t.db.Get(chainStatusKey())
	if data == nil {
		return nil
	}

	chainStatus := &ChainStatus{}
	if err := json.Unmarshal(data, chainStatus); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("get chain status from trace store")
	}

	return chainStatus
}

// SaveChainStatus save the chain status
func (t *TraceStore) SaveChainStatus(chainStatus *ChainStatus) error {
	data, err := json.Marshal(chainStatus)
	if err != nil {
		return err
	}

	t.db.Set(chainStatusKey(), data)
	return nil
}

func (t *TraceStore) saveInstances(instances []*Instance, batch dbm.Batch) error {
	for _, inst := range instances {
		key := instanceKey(inst.TraceID)
		data, err := json.Marshal(inst)
		if err != nil {
			return err
		}

		batch.Set(key, data)
	}
	return nil
}
