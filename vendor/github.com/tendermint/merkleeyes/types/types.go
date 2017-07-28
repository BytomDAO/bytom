package types

import abci "github.com/tendermint/abci/types"

type MerkleEyser interface {
	GetSync(key []byte) abci.Result
	SetSync(key []byte, value []byte) abci.Result
	RemSync(key []byte) abci.Result
	CommitSync() abci.Result
}
