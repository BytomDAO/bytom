package txdb

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

const mainchainPreFix = "MC:"

func calcMainchainKey(hash *bc.Hash) []byte {
	return []byte(mainchainPreFix + hash.String())
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
		mainchain[uint64(i)] = &hash
	}

	return mainchain, nil
}

func saveMainchain(batch dbm.Batch, mainchain map[uint64]*bc.Hash, hash *bc.Hash) error {
	var mainchainList storage.Mainchain
	for i := 0; i < len(mainchain); i++ {
		rawHash := &storage.Mainchain_Hash{Key: mainchain[uint64(i)].Bytes()}
		mainchainList.Hashs = append(mainchainList.Hashs, rawHash)
	}

	b, err := proto.Marshal(&mainchainList)
	if err != nil {
		return errors.Wrap(err, "marshaling Mainchain")
	}

	batch.Set(calcMainchainKey(hash), b)
	return nil
}

func getMainchain(db dbm.DB, hash *bc.Hash) (map[uint64]*bc.Hash, error) {
	data := db.Get(calcMainchainKey(hash))
	if data == nil {
		return nil, errors.New("no this Mainchain")
	}

	mainchain, err := DecodeMainchain(data)
	if err != nil {
		return nil, errors.Wrap(err, "decoding Mainchain")
	}
	return mainchain, nil
}

func cleanMainchainDB(db dbm.DB, hash *bc.Hash) {
	keepKey := calcMainchainKey(hash)

	iter := db.IteratorPrefix([]byte(mainchainPreFix))
	defer iter.Release()
	for iter.Next() {
		if key := iter.Key(); !bytes.Equal(key, keepKey) {
			db.Delete(key)
		}
	}
	db.SetSync(nil, nil)
}
