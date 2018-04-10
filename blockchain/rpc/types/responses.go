package core_types

import (
	"strings"
	"time"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire/data"

	"github.com/bytom/p2p"
	"github.com/bytom/protocol/bc"
)

type BlockNonce [8]byte

type ResultBlockchainInfo struct {
	LastHeight uint64 `json:"last_height"`
}

type ResultBlock struct {
}

type ResultStatus struct {
	NodeInfo          *p2p.NodeInfo `json:"node_info"`
	PubKey            crypto.PubKey `json:"pub_key"`
	LatestBlockHash   data.Bytes    `json:"latest_block_hash"`
	LatestAppHash     data.Bytes    `json:"latest_app_hash"`
	LatestBlockHeight int           `json:"latest_block_height"`
	LatestBlockTime   int64         `json:"latest_block_time"` // nano
}

func (s *ResultStatus) TxIndexEnabled() bool {
	if s == nil || s.NodeInfo == nil {
		return false
	}
	for _, s := range s.NodeInfo.Other {
		info := strings.Split(s, "=")
		if len(info) == 2 && info[0] == "tx_index" {
			return info[1] == "on"
		}
	}
	return false
}

type ResultNetInfo struct {
	Listening bool     `json:"listening"`
	Listeners []string `json:"listeners"`
	Peers     []Peer   `json:"peers"`
}

type ResultBlockHeaderInfo struct {
	Version int32 `json:"version"`
	//Height uint64    `json:"height"`
	MerkleRoot        bc.Hash   `json:"merkleroot"`
	PreviousBlockHash bc.Hash   `json:"prevblockhash"`
	TimestampMS       time.Time `json:"timestamp"`
	Bits              uint64    `json:"bits"`
	Nonce             uint64    `json:"nonce"`
}

type ResultDialSeeds struct {
	Log string `json:"log"`
}

type Peer struct {
	p2p.NodeInfo     `json:"node_info"`
	IsOutbound       bool                 `json:"is_outbound"`
	ConnectionStatus p2p.ConnectionStatus `json:"connection_status"`
}
