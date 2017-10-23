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
)

var latestMainChainStatus = []byte("latestMainChainStatus")

type MainchainStatusJSON struct {
	Height uint64
	Hash   *bc.Hash
}

func (bsj MainchainStatusJSON) Save(db dbm.DB) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		PanicSanity(Fmt("Could not marshal state bytes: %v", err))
	}
	db.SetSync(latestMainChainStatus, bytes)
}

func LoadMainchainStatusJSON(db dbm.DB) MainchainStatusJSON {
	bytes := db.Get(latestMainChainStatus)
	if bytes == nil {
		return MainchainStatusJSON{Height: 0}
	}

	bsj := MainchainStatusJSON{}
	if err := json.Unmarshal(bytes, &bsj); err != nil {
		PanicCrisis(Fmt("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}

func calcMainchainKey(hash *bc.Hash) []byte {
	return []byte(fmt.Sprintf("MC:%v", hash.String()))
}

// DecodeMainchain decodes a Mainchain from bytes
func DecodeMainchain(data []byte) (map[uint64]*bc.Hash, error) {
	var mainchainList storage.Mainchain
	if err := proto.Unmarshal(data, &mainchainList); err != nil {
		return nil, errors.Wrap(err, "unmarshaling Mainchain proto")
	}

	mainchain := make(map[uint64]*bc.Hash)
	for i, rawHash := range mainchainList.Hashs {
		h := &bc.Hash{}
		if err := h.UnmarshalJSON(rawHash.Key); err != nil {
			return nil, errors.Wrap(err, "unmarshaling Mainchain hash")
		}
		mainchain[uint64(i)] = h
	}

	return mainchain, nil
}

func saveMainchain(db dbm.DB, mainchain map[uint64]*bc.Hash, height uint64, hash *bc.Hash) error {
	var mainchainList storage.Mainchain
	for i := 0; i < len(mainchain); i++ {
		rawHash := &storage.Mainchain_Hash{Key: mainchain[uint64(i)].Bytes()}
		mainchainList.Hashs = append(mainchainList.Hashs, rawHash)
	}

	b, err := proto.Marshal(&mainchainList)
	if err != nil {
		return errors.Wrap(err, "marshaling Mainchain")
	}

	// set new Mainchain.
	db.Set(calcMainchainKey(hash), b)
	MainchainStatusJSON{Height: height, Hash: hash}.Save(db)
	db.SetSync(nil, nil)

	//TODO: delete old Mainchain.
	return errors.Wrap(err, "deleting old Mainchains")
}

func getMainchain(db dbm.DB) (map[uint64]*bc.Hash, MainchainStatusJSON, error) {
	mainchainStatus := LoadMainchainStatusJSON(db)
	data := db.Get(calcMainchainKey(mainchainStatus.Hash))
	if data == nil {
		return nil, mainchainStatus, errors.New("no this Mainchain.")
	}

	mainchain, err := DecodeMainchain(data)
	if err != nil {
		return nil, mainchainStatus, errors.Wrap(err, "decoding Mainchain")
	}
	return mainchain, mainchainStatus, nil
}
