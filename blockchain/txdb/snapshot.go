package txdb

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	. "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/internal/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/patricia"
	"github.com/bytom/protocol/state"
)

var latestSnapshotStatus = []byte("latestSnapshotStatus")

type SnapshotStatusJSON struct {
	Height uint64
	Hash   *bc.Hash
}

func (bsj SnapshotStatusJSON) Save(db dbm.DB) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		PanicSanity(Fmt("Could not marshal state bytes: %v", err))
	}
	db.SetSync(latestSnapshotStatus, bytes)
}

func LoadSnapshotStatusJSON(db dbm.DB) SnapshotStatusJSON {
	bytes := db.Get(latestSnapshotStatus)
	if bytes == nil {
		return SnapshotStatusJSON{Height: 0}
	}

	bsj := SnapshotStatusJSON{}
	if err := json.Unmarshal(bytes, &bsj); err != nil {
		PanicCrisis(Fmt("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}

func calcSnapshotKey(hash *bc.Hash) []byte {
	return []byte(fmt.Sprintf("S:%v", hash.String()))
}

// DecodeSnapshot decodes a snapshot from bytes
func DecodeSnapshot(data []byte) (*state.Snapshot, error) {
	var storedSnapshot storage.Snapshot
	if err := proto.Unmarshal(data, &storedSnapshot); err != nil {
		return nil, errors.Wrap(err, "unmarshaling state snapshot proto")
	}

	tree := new(patricia.Tree)
	for _, node := range storedSnapshot.Nodes {
		if err := tree.Insert(node.Key); err != nil {
			return nil, errors.Wrap(err, "reconstructing state tree")
		}
	}

	nonces := make(map[bc.Hash]uint64, len(storedSnapshot.Nonces))
	for _, nonce := range storedSnapshot.Nonces {
		var b32 [32]byte
		copy(b32[:], nonce.Hash)
		hash := bc.NewHash(b32)
		nonces[hash] = nonce.ExpiryMs
	}

	return &state.Snapshot{
		Tree:   tree,
		Nonces: nonces,
	}, nil
}

func saveSnapshot(db dbm.DB, snapshot *state.Snapshot, height uint64, hash *bc.Hash) error {
	var storedSnapshot storage.Snapshot
	err := patricia.Walk(snapshot.Tree, func(key []byte) error {
		n := &storage.Snapshot_StateTreeNode{Key: key}
		storedSnapshot.Nodes = append(storedSnapshot.Nodes, n)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walking patricia tree")
	}

	storedSnapshot.Nonces = make([]*storage.Snapshot_Nonce, 0, len(snapshot.Nonces))
	for k, v := range snapshot.Nonces {
		hash := k
		storedSnapshot.Nonces = append(storedSnapshot.Nonces, &storage.Snapshot_Nonce{
			Hash:     hash.Bytes(), // TODO(bobg): now that hash is a protobuf, use it directly in the snapshot protobuf?
			ExpiryMs: v,
		})
	}

	b, err := proto.Marshal(&storedSnapshot)
	if err != nil {
		return errors.Wrap(err, "marshaling state snapshot")
	}

	// set new snapshot.
	db.Set(calcSnapshotKey(hash), b)
	SnapshotStatusJSON{Height: height, Hash: hash}.Save(db)
	db.SetSync(nil, nil)

	//TODO: delete old snapshot.
	return errors.Wrap(err, "deleting old snapshots")
}

func getSnapshot(db dbm.DB) (*state.Snapshot, SnapshotStatusJSON, error) {
	snapshotStatus := LoadSnapshotStatusJSON(db)
	data := db.Get(calcSnapshotKey(snapshotStatus.Hash))
	if data == nil {
		return nil, snapshotStatus, errors.New("no this snapshot.")
	}

	snapshot, err := DecodeSnapshot(data)
	if err != nil {
		return nil, snapshotStatus, errors.Wrap(err, "decoding snapshot")
	}
	return snapshot, snapshotStatus, nil
}
