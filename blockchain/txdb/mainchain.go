package txdb

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/internal/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

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
		var b32 [32]byte
		copy(b32[:], rawHash.Key)
		hash := bc.NewHash(b32)
		mainchain[uint64(i+1)] = &hash
	}

	return mainchain, nil
}

func saveMainchain(db dbm.DB, mainchain map[uint64]*bc.Hash, hash *bc.Hash) error {
	var mainchainList storage.Mainchain
	for i := 1; i <= len(mainchain); i++ {
		rawHash := &storage.Mainchain_Hash{Key: mainchain[uint64(i)].Bytes()}
		mainchainList.Hashs = append(mainchainList.Hashs, rawHash)
	}

	b, err := proto.Marshal(&mainchainList)
	if err != nil {
		return errors.Wrap(err, "marshaling Mainchain")
	}

	db.Set(calcMainchainKey(hash), b)
	db.SetSync(nil, nil)
	return nil
}

func getMainchain(db dbm.DB, hash *bc.Hash) (map[uint64]*bc.Hash, error) {
	data := db.Get(calcMainchainKey(hash))
	if data == nil {
		return nil, errors.New("no this Mainchain.")
	}

	mainchain, err := DecodeMainchain(data)
	if err != nil {
		return nil, errors.Wrap(err, "decoding Mainchain")
	}
	return mainchain, nil
}
