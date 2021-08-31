package contract

import (
	"bytes"
	"encoding/hex"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/bytom/protocol/bc"
)

type Instance struct {
	TraceID          string
	BlockHeight      uint64
	BlockHash        bc.Hash
	UTXOs            []*UTXO
	UnconfirmedUTXOs map[string][]*UTXO
	ScannedHash      bc.Hash
	Finalized        bool
	InSync           bool
}

type InstanceTable struct {
	idToInst  map[string]*Instance
	keyToInst map[string]*Instance
}

func NewInstanceTable() *InstanceTable {
	return &InstanceTable{
		idToInst:  make(map[string]*Instance),
		keyToInst: make(map[string]*Instance),
	}
}

func (i *InstanceTable) GetByID(id string) *Instance {
	return i.idToInst[id]
}

func (i *InstanceTable) GetByUTXOs(utxos []*UTXO) *Instance {
	return i.keyToInst[utxoKey(utxos)]
}

func (i *InstanceTable) Put(instance *Instance) {
	i.idToInst[instance.TraceID] = instance
	i.keyToInst[utxoKey(instance.UTXOs)] = instance
}

type UTXO struct {
	bc.AssetAmount
	hash    bc.Hash
	program []byte
}

func utxoKey(utxos []*UTXO) string {
	buff := new(bytes.Buffer)
	for _, u := range utxos {
		buff.Write(u.hash.Bytes())
	}
	digest := sha3.Sum256(buff.Bytes())
	return hex.EncodeToString(digest[:])
}
