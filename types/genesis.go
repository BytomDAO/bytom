package types

import (
	"encoding/json"
	"time"

	cmn "github.com/tendermint/tmlibs/common"
)

//------------------------------------------------------------
// core types for a genesis definition
type GenesisDoc struct {
	GenesisTime time.Time  `json:"genesis_time"`
	ChainID     string     `json:"chain_id"`
	PrivateKey  string     `json:"private_key"`
}

// Utility method for saving GenensisDoc as JSON file.
func (genDoc *GenesisDoc) SaveAs(file string) error {
	genDocBytes, err := json.Marshal(genDoc)
	if err != nil {
		return err
	}
	return cmn.WriteFile(file, genDocBytes, 0644)
}

//------------------------------------------------------------
// Make genesis state from file

func GenesisDocFromJSON(jsonBlob []byte) (*GenesisDoc, error) {
	genDoc := GenesisDoc{}
	err := json.Unmarshal(jsonBlob, &genDoc)
	return &genDoc, err
}
