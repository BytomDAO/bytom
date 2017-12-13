package txdb

import (
	"blockchain/testutil"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/state"
)

func TestSaveUtxoView(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	batch := testDB.NewBatch()
	defer os.RemoveAll("temp")

	cases := []struct {
		hash      bc.Hash
		utxoEntry *storage.UtxoEntry
		exist     bool
	}{
		{
			hash:      bc.Hash{V0: 0},
			utxoEntry: storage.NewUtxoEntry(true, 0, true),
			exist:     true,
		},
		{
			hash:      bc.Hash{V0: 1},
			utxoEntry: storage.NewUtxoEntry(true, 0, false),
			exist:     true,
		},
		{
			hash:      bc.Hash{V0: 2},
			utxoEntry: storage.NewUtxoEntry(false, 0, false),
			exist:     true,
		},
		{
			hash:      bc.Hash{V0: 3},
			utxoEntry: storage.NewUtxoEntry(false, 0, true),
			exist:     false,
		},
	}

	view := state.NewUtxoViewpoint()
	for _, c := range cases {
		view.Entries[c.hash] =
			c.utxoEntry
	}

	saveUtxoView(batch, view)
	batch.Write()

	for _, c := range cases {
		entry, err := getUtxo(testDB, &c.hash)

		if !c.exist {
			if err == nil {
				t.Errorf("%v should be unexisted, but it's in the db", c)
			}
			continue
		}

		if !testutil.DeepEqual(entry, c.utxoEntry) {
			t.Errorf("%v utxo in the db isn't match", c)
		}
	}
}
