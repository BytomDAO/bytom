package txdb

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/bytom/blockchain/txdb/internal/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/patricia"
	"github.com/bytom/protocol/state"
	dbm "github.com/tendermint/tmlibs/db"
)

func calcSnapshotKey(height uint64) []byte {
    return []byte(fmt.Sprintf("S:%v", height))
}
/*
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
*/

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
	//TO DO: delete old snapshot.
	db.SetSync(nil, nil)
	return errors.Wrap(err, "deleting old snapshots")
}

/*
func getStateSnapshot(ctx context.Context, db pg.DB) (*state.Snapshot, uint64, error) {
	const q = `
		SELECT data, height FROM snapshots ORDER BY height DESC LIMIT 1
	`
	var (
		data   []byte
		height uint64
	)

	err := db.QueryRowContext(ctx, q).Scan(&data, &height)
	if err == sql.ErrNoRows {
		return state.Empty(), 0, nil
	} else if err != nil {
		return nil, height, errors.Wrap(err, "retrieving state snapshot blob")
	}

	snapshot, err := DecodeSnapshot(data)
	if err != nil {
		return nil, height, errors.Wrap(err, "decoding snapshot")
	}
	return snapshot, height, nil
}

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
