package contract

import (
	"encoding/json"

	dbm "github.com/bytom/bytom/database/leveldb"
)

type TraceStore struct {
	db dbm.DB
}

func NewTraceStore(db dbm.DB) *TraceStore {
	return &TraceStore{db: db}
}

func calcInstanceKey(traceID string) []byte {
	return append([]byte(traceID), []byte(":")...)
}

// GetInstance return instance by given trace id
func (t *TraceStore) GetInstance(traceID string) (*Instance, error) {
	key := calcInstanceKey(traceID)
	data := t.db.Get(key)
	instance := &Instance{}
	if err := json.Unmarshal(data, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// LoadInstances used to load all instances in db
func (t *TraceStore) LoadInstances() ([]*Instance, error) {
	iter := t.db.Iterator()
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
	for _, inst := range instances {
		key := calcInstanceKey(inst.TraceID)
		data, err := json.Marshal(inst)
		if err != nil {
			return err
		}

		batch.Set(key, data)
	}
	batch.Write()
	return nil
}

// RemoveInstance delete a instance by given trace id
func (t *TraceStore) RemoveInstance(traceID string) {
	key := calcInstanceKey(traceID)
	t.db.Delete(key)
}
