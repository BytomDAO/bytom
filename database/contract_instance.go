package database

import (
	"encoding/json"

	"github.com/bytom/bytom/contract"
)

func calcContractInstanceKey(traceID string) []byte {
	return append(checkpointKeyPrefix, []byte(traceID)...)
}

// GetInstance return instance by given trace id
func (s *Store) GetInstance(traceID string) (*contract.Instance, error) {
	key := calcContractInstanceKey(traceID)
	data := s.db.Get(key)
	instance := &contract.Instance{}
	if err := json.Unmarshal(data, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// LoadInstances used to load all instances in db
func (s *Store) LoadInstances() ([]*contract.Instance, error) {
	iter := s.db.Iterator()
	defer iter.Release()

	var instances []*contract.Instance
	for iter.Next() {
		instance := &contract.Instance{}
		if err := json.Unmarshal(iter.Value(), instance); err != nil {
			return nil, err
		}

		instances = append(instances, instance)
	}
	return instances, nil
}

// SaveInstances used to batch save multiple instances
func (s *Store) SaveInstances(instances []*contract.Instance) error {
	batch := s.db.NewBatch()
	for _, inst := range instances {
		key := calcContractInstanceKey(inst.TraceID)
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
func (s *Store) RemoveInstance(traceID string) {
	key := calcContractInstanceKey(traceID)
	s.db.Delete(key)
}
