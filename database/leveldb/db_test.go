package leveldb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTempDB(t *testing.T, backend string) (db DB, dbDir string) {
	dirname, err := ioutil.TempDir("", "db_common_test")
	require.Nil(t, err)
	return NewDB("testdb", backend, dirname), dirname
}

func TestDBIteratorSingleKey(t *testing.T) {
	for backend := range backends {
		t.Run(fmt.Sprintf("Backend %s", backend), func(t *testing.T) {
			db, dir := newTempDB(t, backend)
			defer os.RemoveAll(dir)

			db.Set([]byte("1"), []byte("value_1"))
			itr := db.IteratorPrefixWithStart(nil, nil, false)
			require.Equal(t, []byte(""), itr.Key())
			require.Equal(t, true, itr.Next())
			require.Equal(t, []byte("1"), itr.Key())
		})
	}
}

func TestDBIteratorTwoKeys(t *testing.T) {
	for backend := range backends {
		t.Run(fmt.Sprintf("Backend %s", backend), func(t *testing.T) {
			db, dir := newTempDB(t, backend)
			defer os.RemoveAll(dir)

			db.SetSync([]byte("1"), []byte("value_1"))
			db.SetSync([]byte("2"), []byte("value_1"))

			itr := db.IteratorPrefixWithStart(nil, []byte("1"), false)

			require.Equal(t, []byte("1"), itr.Key())

			require.Equal(t, true, itr.Next())
			itr = db.IteratorPrefixWithStart(nil, []byte("2"), false)

			require.Equal(t, false, itr.Next())
		})
	}
}

func TestDBIterator(t *testing.T) {
	dirname, err := ioutil.TempDir("", "db_common_test")
	require.Nil(t, err)

	db, err := NewGoLevelDB("testdb", dirname)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
		os.RemoveAll(dirname)
	}()

	db.SetSync([]byte("aaa1"), []byte("value_1"))
	db.SetSync([]byte("aaa22"), []byte("value_2"))
	db.SetSync([]byte("bbb22"), []byte("value_3"))

	itr := db.IteratorPrefixWithStart([]byte("aaa"), []byte("aaa1"), false)
	defer itr.Release()

	require.Equal(t, true, itr.Next())
	require.Equal(t, []byte("aaa22"), itr.Key())

	require.Equal(t, false, itr.Next())

	itr = db.IteratorPrefixWithStart([]byte("aaa"), nil, false)

	require.Equal(t, true, itr.Next())
	require.Equal(t, []byte("aaa1"), itr.Key())

	require.Equal(t, true, itr.Next())
	require.Equal(t, []byte("aaa22"), itr.Key())

	require.Equal(t, false, itr.Next())

	itr = db.IteratorPrefixWithStart([]byte("bbb"), []byte("aaa1"), false)
	require.Equal(t, false, itr.Next())
}

func TestDBIteratorReverse(t *testing.T) {
	dirname, err := ioutil.TempDir("", "db_common_test")
	require.Nil(t, err)

	db, err := NewGoLevelDB("testdb", dirname)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
		os.RemoveAll(dirname)
	}()

	db.SetSync([]byte("aaa1"), []byte("value_1"))
	db.SetSync([]byte("aaa22"), []byte("value_2"))
	db.SetSync([]byte("bbb22"), []byte("value_3"))

	itr := db.IteratorPrefixWithStart([]byte("aaa"), []byte("aaa22"), true)
	defer itr.Release()

	require.Equal(t, true, itr.Next())
	require.Equal(t, []byte("aaa1"), itr.Key())

	require.Equal(t, false, itr.Next())

	itr = db.IteratorPrefixWithStart([]byte("aaa"), nil, true)

	require.Equal(t, true, itr.Next())
	require.Equal(t, []byte("aaa22"), itr.Key())

	require.Equal(t, true, itr.Next())
	require.Equal(t, []byte("aaa1"), itr.Key())

	require.Equal(t, false, itr.Next())

	require.Equal(t, false, itr.Next())

	itr = db.IteratorPrefixWithStart([]byte("bbb"), []byte("aaa1"), true)
	require.Equal(t, false, itr.Next())
}
