package contract

import (
	"bytes"
	"encoding/hex"
	"sort"

	"github.com/bytom/bytom/protocol/bc/types"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/bytom/protocol/bc"
)

type Instance struct {
	TraceID   string
	BlockHash bc.Hash
	UTXOs     []*UTXO
	Finalized bool
	InSync    bool
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
	assetID bc.AssetID
	amount  uint64
	hash    bc.Hash
	program []byte
}

func inputToUTXO(input *types.TxInput) *UTXO {
	outputID, _ := input.SpentOutputID()
	return &UTXO{
		assetID: input.AssetID(),
		amount:  input.Amount(),
		hash:    outputID,
		program: input.ControlProgram(),
	}
}

func outputToUTXO(output *types.TxOutput, outputID bc.Hash) *UTXO {
	return &UTXO{
		assetID: *output.AssetId,
		amount:  output.Amount,
		hash:    outputID,
		program: output.ControlProgram,
	}
}

func utxoKey(utxos []*UTXO) string {
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].hash.String() < utxos[j].hash.String()
	})

	buff := new(bytes.Buffer)
	for _, u := range utxos {
		buff.Write(u.hash.Bytes())
	}
	digest := sha3.Sum256(buff.Bytes())
	return hex.EncodeToString(digest[:])
}
