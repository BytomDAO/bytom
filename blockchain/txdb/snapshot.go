package txdb

import (
	"context"
	"fmt"
	"encoding/json"

	"github.com/golang/protobuf/proto"

	"github.com/bytom/blockchain/txdb/internal/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/patricia"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/bc"
	dbm "github.com/tendermint/tmlibs/db"
	. "github.com/tendermint/tmlibs/common"
)

func calcSnapshotKey(height uint64) []byte {
    return []byte(fmt.Sprintf("S:%v", height))
}

func calcLatestSnapshotHeight() []byte {
	return []byte("LatestSnapshotHeight")
}
// DecodeSnapshot decodes a snapshot from the Chain Core's binary,
// protobuf representation of the snapshot.
func DecodeSnapshot(data []byte) (*state.Snapshot, error) {
	var storedSnapshot storage.Snapshot
	err := proto.Unmarshal(data, &storedSnapshot)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling state snapshot proto")
	}

	tree := new(patricia.Tree)
	for _, node := range storedSnapshot.Nodes {
		err = tree.Insert(node.Key)
		if err != nil {
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

var latestSnapshotHeight = []byte("latestSnapshotHeight")

type SnapshotHeightJSON struct {
    Height uint64
}

func (bsj SnapshotHeightJSON) Save(db dbm.DB) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		PanicSanity(Fmt("Could not marshal state bytes: %v", err))
	}
	db.SetSync(latestSnapshotHeight, bytes)
}

func LoadSnapshotHeightJSON(db dbm.DB) SnapshotHeightJSON {
	bytes := db.Get(latestSnapshotHeight)
	if bytes == nil {
		return SnapshotHeightJSON{
			Height: 0,
		}
	}
	bsj := SnapshotHeightJSON{}
	err := json.Unmarshal(bytes, &bsj)
	if err != nil {
		PanicCrisis(Fmt("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}


func storeStateSnapshot(ctx context.Context, db dbm.DB, snapshot *state.Snapshot, blockHeight uint64) error {
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
	db.Set(calcSnapshotKey(blockHeight), b)
	SnapshotHeightJSON{Height: blockHeight}.Save(db)
	//TO DO: delete old snapshot.
	db.SetSync(nil, nil)
	return errors.Wrap(err, "deleting old snapshots")
}

func getStateSnapshot(ctx context.Context, db dbm.DB) (*state.Snapshot, uint64, error) {
	height := LoadSnapshotHeightJSON(db).Height
	data := db.Get(calcSnapshotKey(height))
	if data == nil {
		return nil, height, errors.New("no this snapshot.")
	}

	snapshot, err := DecodeSnapshot(data)
	if err != nil {
		return nil, height, errors.Wrap(err, "decoding snapshot")
	}
	return snapshot, height, nil
}

/*
// getRawSnapshot returns the raw, protobuf-encoded snapshot data at the
// provided height.
func getRawSnapshot(ctx context.Context, db pg.DB, height uint64) (data []byte, err error) {
	const q = `SELECT data FROM snapshots WHERE height = $1`
	err = db.QueryRowContext(ctx, q, height).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	}
	return data, err
}
*/
