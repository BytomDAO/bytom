package app

import (
	"fmt"
	"path"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/merkleeyes/iavl"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/merkle"
)

type MerkleEyesApp struct {
	abci.BaseApplication

	state  State
	db     dbm.DB
	height uint64
}

// just make sure we really are an application, if the interface
// ever changes in the future
func (m *MerkleEyesApp) assertApplication() abci.Application {
	return m
}

var eyesStateKey = []byte("merkleeyes:state") // Database key for merkle tree save value db values

type MerkleEyesState struct {
	Hash   []byte
	Height uint64
}

const (
	WriteSet byte = 0x01
	WriteRem byte = 0x02
)

func NewMerkleEyesApp(dbName string, cacheSize int) *MerkleEyesApp {
	// start at 1 so the height returned by query is for the
	// next block, ie. the one that includes the AppHash for our current state
	initialHeight := uint64(1)

	// Non-persistent case
	if dbName == "" {
		tree := iavl.NewIAVLTree(
			0,
			nil,
		)
		return &MerkleEyesApp{
			state:  NewState(tree, false),
			db:     nil,
			height: initialHeight,
		}
	}

	// Setup the persistent merkle tree
	empty, _ := cmn.IsDirEmpty(path.Join(dbName, dbName+".db"))

	// Open the db, if the db doesn't exist it will be created
	db := dbm.NewDB(dbName, dbm.LevelDBBackendStr, dbName)

	// Load Tree
	tree := iavl.NewIAVLTree(cacheSize, db)

	if empty {
		fmt.Println("no existing db, creating new db")
		db.Set(eyesStateKey, wire.BinaryBytes(MerkleEyesState{
			Hash:   tree.Save(),
			Height: initialHeight,
		}))
	} else {
		fmt.Println("loading existing db")
	}

	// Load merkle state
	eyesStateBytes := db.Get(eyesStateKey)
	var eyesState MerkleEyesState
	err := wire.ReadBinaryBytes(eyesStateBytes, &eyesState)
	if err != nil {
		fmt.Println("error reading MerkleEyesState")
		panic(err.Error())
	}
	tree.Load(eyesState.Hash)

	return &MerkleEyesApp{
		state:  NewState(tree, true),
		db:     db,
		height: eyesState.Height,
	}
}

func (app *MerkleEyesApp) CloseDB() {
	if app.db != nil {
		app.db.Close()
	}
}

func (app *MerkleEyesApp) Info() abci.ResponseInfo {
	return abci.ResponseInfo{Data: cmn.Fmt("size:%v", app.state.Committed().Size())}
}

func (app *MerkleEyesApp) SetOption(key string, value string) (log string) {
	return "No options are supported yet"
}

func (app *MerkleEyesApp) DeliverTx(tx []byte) abci.Result {
	tree := app.state.Append()
	return app.doTx(tree, tx)
}

func (app *MerkleEyesApp) CheckTx(tx []byte) abci.Result {
	tree := app.state.Check()
	return app.doTx(tree, tx)
}

func (app *MerkleEyesApp) doTx(tree merkle.Tree, tx []byte) abci.Result {
	if len(tx) == 0 {
		return abci.ErrEncodingError.SetLog("Tx length cannot be zero")
	}
	typeByte := tx[0]
	tx = tx[1:]
	switch typeByte {
	case WriteSet: // Set
		key, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return abci.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		}
		tx = tx[n:]
		value, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return abci.ErrEncodingError.SetLog(cmn.Fmt("Error reading value: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return abci.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		}

		tree.Set(key, value)
		// fmt.Println("SET", cmn.Fmt("%X", key), cmn.Fmt("%X", value))
	case WriteRem: // Remove
		key, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return abci.ErrEncodingError.SetLog(cmn.Fmt("Error reading key: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return abci.ErrEncodingError.SetLog(cmn.Fmt("Got bytes left over"))
		}
		tree.Remove(key)
	default:
		return abci.ErrUnknownRequest.SetLog(cmn.Fmt("Unexpected Tx type byte %X", typeByte))
	}
	return abci.OK
}

func (app *MerkleEyesApp) Commit() abci.Result {

	hash := app.state.Commit()

	app.height++
	if app.db != nil {
		app.db.Set(eyesStateKey, wire.BinaryBytes(MerkleEyesState{
			Hash:   hash,
			Height: app.height,
		}))
	}

	if app.state.Committed().Size() == 0 {
		return abci.NewResultOK(nil, "Empty hash for empty tree")
	}
	return abci.NewResultOK(hash, "")
}

func (app *MerkleEyesApp) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	if len(reqQuery.Data) == 0 {
		return
	}
	tree := app.state.Committed()

	if reqQuery.Height != 0 {
		// TODO: support older commits
		resQuery.Code = abci.CodeType_InternalError
		resQuery.Log = "merkleeyes only supports queries on latest commit"
		return
	}

	// set the query response height
	resQuery.Height = app.height

	switch reqQuery.Path {
	case "/store", "/key": // Get by key
		key := reqQuery.Data // Data holds the key bytes
		resQuery.Key = key
		if reqQuery.Prove {
			value, proof, exists := tree.Proof(key)
			if !exists {
				resQuery.Log = "Key not found"
			}
			resQuery.Value = value
			resQuery.Proof = proof
			// TODO: return index too?
		} else {
			index, value, _ := tree.Get(key)
			resQuery.Value = value
			resQuery.Index = int64(index)
		}

	case "/index": // Get by Index
		index := wire.GetInt64(reqQuery.Data)
		key, value := tree.GetByIndex(int(index))
		resQuery.Key = key
		resQuery.Index = int64(index)
		resQuery.Value = value

	case "/size": // Get size
		size := tree.Size()
		sizeBytes := wire.BinaryBytes(size)
		resQuery.Value = sizeBytes

	default:
		resQuery.Code = abci.CodeType_UnknownRequest
		resQuery.Log = cmn.Fmt("Unexpected Query path: %v", reqQuery.Path)
	}
	return
}
