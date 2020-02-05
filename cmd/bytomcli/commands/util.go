package commands

import (
	stdjson "encoding/json"
	"os"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/encoding/json"
	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/util"
)

// accountIns is used for account related request.
type accountIns struct {
	RootXPubs   []chainkd.XPub `json:"root_xpubs"`
	Quorum      int            `json:"quorum"`
	Alias       string         `json:"alias"`
	AccessToken string         `json:"access_token"`
}

// assetIns is used for asset related request.
type assetIns struct {
	RootXPubs       []chainkd.XPub         `json:"root_xpubs"`
	Quorum          int                    `json:"quorum"`
	Alias           string                 `json:"alias"`
	Definition      map[string]interface{} `json:"definition"`
	IssuanceProgram chainjson.HexBytes     `json:"issuance_program"`
	AccessToken     string                 `json:"access_token"`
}

type requestQuery struct {
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
	PageSize     int           `json:"page_size"`
	AscLongPoll  bool          `json:"ascending_with_long_poll,omitempty"`
	Timeout      json.Duration `json:"timeout"`
	After        string        `json:"after"`
	StartTimeMS  uint64        `json:"start_time,omitempty"`
	EndTimeMS    uint64        `json:"end_time,omitempty"`
	TimestampMS  uint64        `json:"timestamp,omitempty"`
	Type         string        `json:"type"`
	Aliases      []string      `json:"aliases,omitempty"`
}

// txFeed
type txFeed struct {
	Alias  string `json:"alias"`
	Filter string `json:"filter,omitempty"`
}

type respArrayTxFeed struct {
	Status string    `json:"status,omitempty"`
	Msg    string    `json:"msg,omitempty"`
	Data   []*txFeed `json:"data,omitempty"`
}

type respTxFeed struct {
	Status string `json:"status,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Data   txFeed `json:"data,omitempty"`
}

type accessToken struct {
	ID      string    `json:"id,omitempty"`
	Token   string    `json:"token,omitempty"`
	Type    string    `json:"type,omitempty"`
	Secret  string    `json:"secret,omitempty"`
	Created time.Time `json:"created_at,omitempty"`
}

func printJSON(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if ok != true {
		jww.ERROR.Println("invalid type assertion")
		os.Exit(util.ErrLocalParse)
	}

	rawData, err := stdjson.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		jww.ERROR.Println(err)
		os.Exit(util.ErrLocalParse)
	}

	jww.FEEDBACK.Println(string(rawData))
}

func printJSONList(data interface{}) {
	dataList, ok := data.([]interface{})
	if ok != true {
		jww.ERROR.Println("invalid type assertion")
		os.Exit(util.ErrLocalParse)
	}

	for idx, item := range dataList {
		jww.FEEDBACK.Println(idx, ":")
		rawData, err := stdjson.MarshalIndent(item, "", "  ")
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalParse)
		}

		jww.FEEDBACK.Println(string(rawData))
	}
}
