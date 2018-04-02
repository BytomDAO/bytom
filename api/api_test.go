package api

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/accesstoken"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/testutil"
)

func TestAPIHandler(t *testing.T) {
	// the TestResponse is same with api Response
	type TestResponse struct {
		Status string      `json:"status,omitempty"`
		Msg    string      `json:"msg,omitempty"`
		Data   interface{} `json:"data,omitempty"`
	}

	a := &API{}
	response := &TestResponse{}

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
		path        string
		request     interface{}
		respWant    *TestResponse
		respWantErr error
	}{
		{
			path: "/create-key",
			request: struct {
				Alias    string `json:"alias"`
				Password string `json:"password"`
			}{Alias: "alice", Password: "123456"},
			respWant: &TestResponse{
				Status: "fail",
				Msg:    "wallet not found, please check that the wallet is open",
			},
		},
		{
			path:    "/error",
			request: nil,
			respWant: &TestResponse{
				Status: "fail",
				Msg:    "wallet not found, please check that the wallet is open",
			},
		},
		{
			path:     "/",
			request:  nil,
			respWant: &TestResponse{Status: "", Msg: "", Data: interface{}(nil)},
		},
		{
			path: "/create-access-token",
			request: struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}{ID: "test-access-id", Type: "test-access-type"},
			respWant: &TestResponse{
				Status: "success",
				Msg:    "",
				Data:   map[string]interface{}{"id": "test-access-id", "type": "test-access-type", "token": "test-access-id:440d87ae0d625a7fcf076275b18372e09a0899e37ec86398879388de90cb0c67"},
			},
		},
		{
			path:    "/gas-rate",
			request: nil,
			respWant: &TestResponse{
				Status: "success",
				Msg:    "",
				Data:   map[string]interface{}{"gasRate": 1000},
			},
		},
	}

	for _, c := range cases {
		response = &TestResponse{}
		client.Call(context.Background(), c.path, c.request, &response)

		if !testutil.DeepEqual(response.Status, c.respWant.Status) {
			t.Errorf(`got=%#v; want=%#v`, response.Status, c.respWant.Status)
		}
	}
}
