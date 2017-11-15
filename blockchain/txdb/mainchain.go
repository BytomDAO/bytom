package txdb

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/internal/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"hash"
)

func calcMainchainKey(hash *bc.Hash) []byte {
	return []byte(fmt.Sprintf("MC:%v", hash.String()))
}

const RollBackPreFix = "RB:"

func calcRollBackKey(hash *bc.Hash) []byte {
	return []byte(RollBackPreFix + hash.String())
}
func calcRBKeyString(hash string) []byte {
	return []byte(RollBackPreFix + hash)
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

func DecodeRollBack(data []byte) (*bc.Rollback, error) {
	var rbList storage.Rollback
	if err := proto.Unmarshal(data, &rbList); err != nil {
		return nil, errors.Wrap(err, "unmarshaling RollBack proto")
	}

	rollBackList := bc.Rollback{Detach: make([]*bc.Hash, 0), Attach: make([]*bc.Hash, 0)}
	for _, rawHash := range rbList.Detach {
		var b32 [32]byte
		copy(b32[:], rawHash.Key)
		hash := bc.NewHash(b32)
		rollBackList.Detach = append(rollBackList.Detach, &hash)
	}
	for _, rawHash := range rbList.Attach {
		var b32 [32]byte
		copy(b32[:], rawHash.Key)
		hash := bc.NewHash(b32)
		rollBackList.Attach = append(rollBackList.Attach, &hash)
	}

	return &rollBackList, nil
}

func saveMainchain(db dbm.DB, mainchain map[uint64]*bc.Hash, hash *bc.Hash, rollback *bc.Rollback) error {
	var mainchainList storage.Mainchain
	storeBatch := db.NewBatch()
	for i := 1; i <= len(mainchain); i++ {
		rawHash := &storage.Mainchain_Hash{Key: mainchain[uint64(i)].Bytes()}
		mainchainList.Hashs = append(mainchainList.Hashs, rawHash)
	}

	b, err := proto.Marshal(&mainchainList)
	if err != nil {
		return errors.Wrap(err, "marshaling Mainchain")
	}

	storeBatch.Set(calcMainchainKey(hash), b)

	var rbList storage.Rollback
	for i := 1; i <= len(rollback.Detach); i++ {
		rawHash := &storage.Rollback_Hash{Key: rollback.Detach[i].Bytes()}
		rbList.Detach = append(rbList.Detach, rawHash)
	}
	for i := 1; i <= len(rollback.Attach); i++ {
		rawHash := &storage.Rollback_Hash{Key: rollback.Attach[i].Bytes()}
		rbList.Detach = append(rbList.Attach, rawHash)
	}

	r, err := proto.Marshal(&rbList)
	if err != nil {
		return errors.Wrap(err, "marshaling Rollback")
	}

	storeBatch.Set(calcRollBackKey(hash), r)

	storeBatch.Write()

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

func getRollBackMap(db dbm.DB) (map[string]*bc.Rollback, error) {
	rollBackMap := make(map[string]*bc.Rollback, 0)
	rollBackIter := db.IteratorPrefix([]byte(RollBackPreFix))
	defer rollBackIter.Release()

	for rollBackIter.Next() {
		rollBack, err := DecodeRollBack(rollBackIter.Value())
		if err != nil {
			return nil, errors.Wrap(err, "decoding Mainchain")
		}
		hash := string(rollBackIter.Key())[len(RollBackPreFix):]
		rollBackMap[hash] = rollBack
	}

	return rollBackMap, nil
}

func delRollBack(db dbm.DB, rbHash string) {
	db.Delete(calcRBKeyString(rbHash))
}
