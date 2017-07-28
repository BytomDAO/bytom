package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/abci/types"
	wire "github.com/tendermint/go-wire"
	"github.com/tendermint/merkleeyes/iavl"
)

func makeSet(key, value []byte) []byte {
	tx := make([]byte, 1+wire.ByteSliceSize(key)+wire.ByteSliceSize(value))
	buf := tx
	buf[0] = WriteSet // Set TypeByte
	buf = buf[1:]
	n, err := wire.PutByteSlice(buf, key)
	if err != nil {
		panic(err)
	}
	buf = buf[n:]
	n, err = wire.PutByteSlice(buf, value)
	if err != nil {
		panic(err)
	}
	return tx
}

func makeRemove(key []byte) []byte {
	tx := make([]byte, 1+wire.ByteSliceSize(key))
	buf := tx
	buf[0] = WriteRem // Set TypeByte
	buf = buf[1:]
	_, err := wire.PutByteSlice(buf, key)
	if err != nil {
		panic(err)
	}
	return tx
}

func makeQuery(key []byte, prove bool, height uint64) (reqQuery abci.RequestQuery) {
	reqQuery.Path = "/key"
	reqQuery.Data = key
	reqQuery.Prove = prove
	reqQuery.Height = height
	return
}

func TestAppQueries(t *testing.T) {
	assert := assert.New(t)

	app := NewMerkleEyesApp("", 0)
	info := app.Info().Data
	assert.Equal("size:0", info)
	com := app.Commit()
	assert.EqualValues([]byte(nil), com.Data)

	// prepare some actions
	key, value := []byte("foobar"), []byte("works!")
	addTx := makeSet(key, value)
	removeTx := makeRemove(key)

	// need to commit append before it shows in queries
	append := app.DeliverTx(addTx)
	assert.True(append.IsOK(), append.Log)
	info = app.Info().Data
	assert.Equal("size:0", info)
	resQuery := app.Query(makeQuery(key, false, 0))
	assert.True(resQuery.Code.IsOK(), resQuery.Log)
	assert.Equal([]byte(nil), resQuery.Value)

	com = app.Commit()
	hash := com.Data
	assert.NotEqual(t, nil, hash)
	info = app.Info().Data
	assert.Equal("size:1", info)
	resQuery = app.Query(makeQuery(key, false, 0))
	assert.True(resQuery.Code.IsOK(), resQuery.Log)
	assert.Equal(value, resQuery.Value)

	// modifying check has no effect
	check := app.CheckTx(removeTx)
	assert.True(check.IsOK(), check.Log)
	com = app.Commit()
	assert.True(com.IsOK(), com.Log)
	hash2 := com.Data
	assert.Equal(hash, hash2)
	info = app.Info().Data
	assert.Equal("size:1", info)

	// proofs come from the last commited state, not working state
	append = app.DeliverTx(removeTx)
	assert.True(append.IsOK(), append.Log)
	// currently don't support specifying block height
	resQuery = app.Query(makeQuery(key, true, 1))
	assert.False(resQuery.Code.IsOK(), resQuery.Log)
	resQuery = app.Query(makeQuery(key, true, 0))
	if assert.NotEmpty(resQuery.Value) {
		proof, err := iavl.ReadProof(resQuery.Proof)
		if assert.Nil(err) {
			assert.True(proof.Verify(key, resQuery.Value, proof.RootHash))
		}
	}

	// commit remove actually removes it now
	com = app.Commit()
	assert.True(com.IsOK(), com.Log)
	hash3 := com.Data
	assert.NotEqual(hash, hash3)
	info = app.Info().Data
	assert.Equal("size:0", info)

	// nothing here...
	resQuery = app.Query(makeQuery(key, false, 0))
	assert.True(resQuery.Code.IsOK(), resQuery.Log)
	assert.Equal([]byte(nil), resQuery.Value)
	// neither with proof...
	resQuery = app.Query(makeQuery(key, true, 0))
	assert.True(resQuery.Code.IsOK(), resQuery.Log)
	assert.Equal([]byte(nil), resQuery.Value)
	assert.Empty(resQuery.Proof)
}
