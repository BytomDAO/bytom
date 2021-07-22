package database

import (
	"encoding/binary"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
)

func calcCheckpointKey(height uint64, hash *bc.Hash) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	key := append(checkpointKeyPrefix, buf...)
	if hash != nil {
		key = append(key, hash.Bytes()...)
	}
	return key
}

func getCheckpointFromDB(db dbm.DB, key []byte) (*state.Checkpoint, error) {
	checkpoint := &state.Checkpoint{}
	if err := json.Unmarshal(db.Get(key), checkpoint); err != nil {
		return nil, err
	}

	return checkpoint, nil
}

func (s *Store) GetCheckpoint(hash *bc.Hash) (*state.Checkpoint, error) {
	header, err := s.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	checkpoint, err := s.cache.lookupCheckPoint(calcCheckpointKey(header.Height, hash))
	if err != nil {
		return nil, err
	}

	checkpoint.SupLinks = append(checkpoint.SupLinks, header.SupLinks...)
	return checkpoint, nil
}

// GetCheckpointsByHeight return all checkpoints of specified block height
func (s *Store) GetCheckpointsByHeight(height uint64) ([]*state.Checkpoint, error) {
	iter := s.db.IteratorPrefix(calcCheckpointKey(height, nil))
	defer iter.Release()
	return s.loadCheckpointsFromIter(iter)
}

// CheckpointsFromNode return all checkpoints from specified block height and hash
func (s *Store) CheckpointsFromNode(height uint64, hash *bc.Hash) ([]*state.Checkpoint, error) {
	startKey := calcCheckpointKey(height, hash)
	iter := s.db.IteratorPrefixWithStart(checkpointKeyPrefix, startKey, false)

	firstCheckpoint := &state.Checkpoint{}
	if err := json.Unmarshal(iter.Value(), firstCheckpoint); err != nil {
		return nil, err
	}

	checkpoints := []*state.Checkpoint{firstCheckpoint}
	subs, err := s.loadCheckpointsFromIter(iter)
	if err != nil {
		return nil, err
	}

	checkpoints = append(checkpoints, subs...)
	return checkpoints, nil
}

func (s *Store) loadCheckpointsFromIter(iter dbm.Iterator) ([]*state.Checkpoint, error) {
	var checkpoints []*state.Checkpoint
	defer iter.Release()
	for iter.Next() {
		checkpoint := &state.Checkpoint{}
		if err := json.Unmarshal(iter.Value(), checkpoint); err != nil {
			return nil, err
		}

		header, err := s.GetBlockHeader(&checkpoint.Hash)
		if err != nil {
			return nil, err
		}

		checkpoint.SupLinks = append(checkpoint.SupLinks, header.SupLinks...)
		checkpoints = append(checkpoints, checkpoint)
	}
	return checkpoints, nil
}

// SaveCheckpoints bulk save multiple checkpoint
func (s *Store) SaveCheckpoints(checkpoints []*state.Checkpoint) error {
	var keys [][]byte

	batch := s.db.NewBatch()
	for _, checkpoint := range checkpoints {
		startTime := time.Now()
		data, err := json.Marshal(checkpoint)
		if err != nil {
			return err
		}

		key := calcCheckpointKey(checkpoint.Height, &checkpoint.Hash)
		batch.Set(key, data)
		keys = append(keys, key)
		log.WithFields(log.Fields{
			"module":   logModule,
			"height":   checkpoint.Height,
			"hash":     checkpoint.Hash.String(),
			"status":   checkpoint.Status,
			"duration": time.Since(startTime),
		}).Info("checkpoint saved on disk")
	}

	batch.Write()

	for _, key := range keys {
		s.cache.removeCheckPoint(key)
	}

	return nil
}
