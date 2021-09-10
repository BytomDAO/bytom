package contract

import (
	"encoding/json"

	dbm "github.com/bytom/bytom/database/leveldb"
)

type InstanceDB struct {
	db dbm.DB
}

func calcInstanceKey(traceID string) []byte {
	return append([]byte(traceID), []byte(":")...)
}

// GetInstance return instance by given trace id
func (i *InstanceDB) GetInstance(traceID string) (*Instance, error) {
	key := calcInstanceKey(traceID)
	data := i.db.Get(key)
	instance := &Instance{}
	if err := json.Unmarshal(data, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// LoadInstances used to load all instances in db
func (i *InstanceDB) LoadInstances() ([]*Instance, error) {
	iter := i.db.Iterator()
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
func (i *InstanceDB) SaveInstances(instances []*Instance) error {
	batch := i.db.NewBatch()
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
func (i *InstanceDB) RemoveInstance(traceID string) {
	key := calcInstanceKey(traceID)
	i.db.Delete(key)
}
