package api

import (
	"context"
	"encoding/json"
	"math"
	"net/http/httptest"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/accesstoken"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/testutil"
)

func TestAPIHandler(t *testing.T) {
	a := &API{}
	response := &Response{}

	// init httptest server
	a.buildHandler()
	server := httptest.NewServer(a.handler)
	defer server.Close()

	// create accessTokens
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	a.accessTokens = accesstoken.NewStore(testDB)

	client := &rpc.Client{
		BaseURL:     server.URL,
		AccessToken: "test-user:test-secret",
	}

	cases := []struct {
		path     string
		request  interface{}
		respWant *Response
	}{
		{
			path: "/create-key",
			request: struct {
				Alias    string `json:"alias"`
				Password string `json:"password"`
			}{Alias: "alice", Password: "123456"},
			respWant: &Response{
				Status: "fail",
				Msg:    "wallet not found, please check that the wallet is open",
			},
		},
		{
			path:    "/error",
			request: nil,
			respWant: &Response{
				Status: "fail",
				Msg:    "wallet not found, please check that the wallet is open",
			},
		},
		{
			path:    "/",
			request: nil,
			respWant: &Response{
				Status: "",
				Msg:    "",
			},
		},
		{
			path: "/create-access-token",
			request: struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}{ID: "test-access-id", Type: "test-access-type"},
			respWant: &Response{
				Status: "success",
				Msg:    "",
				Data:   map[string]interface{}{"id": "test-access-id", "type": "test-access-type", "token": "test-access-id:440d87ae0d625a7fcf076275b18372e09a0899e37ec86398879388de90cb0c67"},
			},
		},
		{
			path:    "/gas-rate",
			request: nil,
			respWant: &Response{
				Status: "success",
				Msg:    "",
				Data:   map[string]interface{}{"gasRate": 1000},
			},
		},
	}

	for _, c := range cases {
		response = &Response{}
		client.Call(context.Background(), c.path, c.request, &response)

		if !testutil.DeepEqual(response.Status, c.respWant.Status) {
			t.Errorf(`got=%#v; want=%#v`, response.Status, c.respWant.Status)
		}
	}
}

func TestEstimateTxGas(t *testing.T) {
	tmplStr := `{"allow_additional_actions":false,"raw_transaction":"070100010161015ffe8a1209937a6a8b22e8c01f056fd5f1730734ba8964d6b79de4a639032cecddffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8099c4d59901000116001485eb6eee8023332da85df60157dc9b16cc553fb2010002013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80afa08b4f011600142b4fd033bc76b4ddf5cb00f625362c4bc7b10efa00013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8090dfc04a011600146eea1ce6cfa5b718ae8094376be9bc1a87c9c82700","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0100000000000000"],"xpub":"cb4e5932d808ee060df9552963d87f60edac42360b11d4ad89558ef2acea4d4aaf4818f2ebf5a599382b8dfce0a0c798c7e44ec2667b3a1d34c61ba57609de55"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"1c9b5c1db7f4afe31fd1b7e0495a8bb042a271d8d7924d4fc1ff7cf1bff15813"}]}]}`
	template := txbuilder.Template{}
	err := json.Unmarshal([]byte(tmplStr), &template)
	if err != nil {
		t.Fatal(err)
	}

	estimateResult, err := EstimateTxGas(template)
	if err != nil {
		t.Fatal(err)
	}

	baseRate := float64(100000)
	totalNeu := float64(estimateResult.StorageNeu+estimateResult.VMNeu) / baseRate
	roundingNeu := math.Ceil(totalNeu)
	estimateNeu := int64(roundingNeu) * int64(baseRate)

	if estimateResult.TotalNeu != estimateNeu {
		t.Errorf(`got=%#v; want=%#v`, estimateResult.TotalNeu, estimateNeu)
	}
}
