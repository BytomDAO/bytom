package leveldb

import (
	"bytes"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestSaveMainchain(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	inputMain := make(map[uint64]*bc.Hash)
	for i := uint64(0); i <= uint64(10); i++ {
		inputMain[i] = &bc.Hash{V0: i}
	}

	saveHash := &bc.Hash{V0: 0}
	batch := testDB.NewBatch()
	if err := saveMainchain(batch, inputMain, saveHash); err != nil {
		t.Errorf(err.Error())
	}
	batch.Write()

	fetchMain, err := getMainchain(testDB, saveHash)
	if err != nil {
		t.Errorf(err.Error())
	}

	if !testutil.DeepEqual(inputMain, fetchMain) {
		t.Errorf("inputMain and fetchMain is not equal")
	}
}

func TestCleanMainchainDB(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	// Insert the test data
	hash := &bc.Hash{}
	for i := uint64(0); i <= uint64(10); i++ {
		hash.V0 = i
		testDB.Set(calcMainchainKey(hash), nil)
	}
	testDB.SetSync(nil, nil)

	// run the test function
	cleanMainchainDB(testDB, hash)

	// check the clean result
	iter := testDB.IteratorPrefix([]byte(mainchainPreFix))
	defer iter.Release()

	if !iter.Next() || !bytes.Equal(iter.Key(), calcMainchainKey(hash)) {
		t.Errorf("latest mainchain get deleted from db")
	}
	if iter.Next() {
		t.Errorf("more than one mainchain still saved in the db")
	}
}
