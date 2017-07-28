package eyes

import (
	abcicli "github.com/tendermint/abci/client"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/merkleeyes/app"
)

type Client struct {
	abcicli.Client
}

func NewClient(addr string) (*Client, error) {
	abciClient, err := abcicli.NewClient(addr, "socket", false)
	if err != nil {
		return nil, err
	}
	client := &Client{
		Client: abciClient,
	}
	return client, nil
}

func NewLocalClient(dbName string, cacheSize int) *Client {
	eyesApp := app.NewMerkleEyesApp(dbName, cacheSize)
	abciClient := abcicli.NewLocalClient(nil, eyesApp)
	return &Client{
		Client: abciClient,
	}
}

// Convenience KVStore interface
func (client *Client) Get(key []byte) (value []byte) {
	_, value, err := client.GetByKey(key)
	if err != nil {
		panic("requesting ABCI Query: " + err.Error())
	}
	return value
}

func (client *Client) GetByKey(key []byte) (index int64, value []byte, err error) {
	resQuery, err := client.QuerySync(abci.RequestQuery{
		Path:   "/key",
		Data:   key,
		Height: 0,
	})
	if err != nil {
		return
	}
	return resQuery.Index, resQuery.Value, nil
}

// TODO: Support returning index too?
func (client *Client) GetByKeyWithProof(key []byte) (value []byte, proof []byte, err error) {
	resQuery, err := client.QuerySync(abci.RequestQuery{
		Path:   "/key",
		Data:   key,
		Height: 0,
		Prove:  true,
	})
	if err != nil {
		return
	}
	return resQuery.Value, resQuery.Proof, nil
}

func (client *Client) GetByIndex(index int64) (key []byte, value []byte, err error) {
	resQuery, err := client.QuerySync(abci.RequestQuery{
		Path:   "/index",
		Data:   wire.BinaryBytes(index),
		Height: 0,
	})
	if err != nil {
		return
	}
	return resQuery.Key, resQuery.Value, nil
}

// Convenience KVStore interface
func (client *Client) Set(key []byte, value []byte) {
	tx := make([]byte, 1+wire.ByteSliceSize(key)+wire.ByteSliceSize(value))
	buf := tx
	buf[0] = app.WriteSet // Set TypeByte
	buf = buf[1:]
	n, err := wire.PutByteSlice(buf, key)
	if err != nil {
		panic("encoding key byteslice: " + err.Error())
	}
	buf = buf[n:]
	n, err = wire.PutByteSlice(buf, value)
	if err != nil {
		panic("encoding value byteslice: " + err.Error())
	}
	res := client.DeliverTxSync(tx)
	if res.IsErr() {
		panic(res.Error())
	}
}

// Convenience
func (client *Client) Remove(key []byte) {
	tx := make([]byte, 1+wire.ByteSliceSize(key))
	buf := tx
	buf[0] = app.WriteRem // Rem TypeByte
	buf = buf[1:]
	_, err := wire.PutByteSlice(buf, key)
	if err != nil {
		panic("encoding key byteslice: " + err.Error())
	}
	res := client.DeliverTxSync(tx)
	if res.IsErr() {
		panic(res.Error())
	}
}
