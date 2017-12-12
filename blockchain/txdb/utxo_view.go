package txdb

import (
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/txdb/storage"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/state"
	"github.com/golang/protobuf/proto"
)

const utxoPreFix = "UT:"

func calcUtxoKey(hash *bc.Hash) []byte {
	return []byte(utxoPreFix + hash.String())
}

func getBlockUtxos(db dbm.DB, view *state.UtxoViewpoint, block *bc.Block) error {
	var utxo storage.UtxoEntry
	for _, tx := range block.Transactions {
		for _, prevout := range tx.SpentOutputIDs {
			data := db.Get(calcUtxoKey(&prevout))
			if data == nil {
				return errors.New("can't find utxo in db")
			}

			if err := proto.Unmarshal(data, &utxo); err != nil {
				return errors.Wrap(err, "unmarshaling utxo entry")
			}

			view.Entries[prevout] = &utxo
		}
	}

	return nil
}

func saveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	for key, entry := range view.Entries {
		if entry.Spend && !entry.IsCoinBase {
			batch.Delete(calcUtxoKey(&key))
			continue
		}

		b, err := proto.Marshal(entry)
		if err != nil {
			return errors.Wrap(err, "marshaling utxo entry")
		}
		batch.Set(calcUtxoKey(&key), b)
	}
	return nil
}
