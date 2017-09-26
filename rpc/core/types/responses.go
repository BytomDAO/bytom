package core_types

import (
	"strings"
    "time"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire/data"
    "github.com/bytom/protocol/bc"
	"github.com/bytom/p2p"
	"github.com/bytom/types"
)

type BlockNonce [8]byte

type ResultBlockchainInfo struct {
	LastHeight uint64                `json:"last_height"`
}

type ResultGenesis struct {
	Genesis *types.GenesisDoc `json:"genesis"`
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
        Version int32   `json:"version"`
        //Height uint64    `json:"height"`
        MerkleRoot bc.Hash  `json:"merkleroot"`
        PreviousBlockHash bc.Hash  `json:"prevblockhash"`
        TimestampMS time.Time   `json:"timestamp"`
        Bits uint64      `json:"bits"`
        Nonce uint64     `json:"nonce"`
}

type ResultDialSeeds struct {
	Log string `json:"log"`
}

type Peer struct {
	p2p.NodeInfo     `json:"node_info"`
	IsOutbound       bool                 `json:"is_outbound"`
	ConnectionStatus p2p.ConnectionStatus `json:"connection_status"`
}

type ResultDumpConsensusState struct {
	RoundState      string   `json:"round_state"`
	PeerRoundStates []string `json:"peer_round_states"`
}

type ResultBroadcastTx struct {
	Code abci.CodeType `json:"code"`
	Data data.Bytes    `json:"data"`
	Log  string        `json:"log"`

	Hash data.Bytes `json:"hash"`
}

type ResultBroadcastTxCommit struct {
	CheckTx   abci.Result `json:"check_tx"`
	DeliverTx abci.Result `json:"deliver_tx"`
	Hash      data.Bytes  `json:"hash"`
	Height    int         `json:"height"`
}

type ResultABCIInfo struct {
	Response abci.ResponseInfo `json:"response"`
}

type ResultABCIQuery struct {
	*abci.ResultQuery `json:"response"`
}

type ResultUnsafeFlushMempool struct{}

type ResultUnsafeProfile struct{}

type ResultSubscribe struct{}

type ResultUnsubscribe struct{}
