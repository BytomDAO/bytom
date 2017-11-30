package authn

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/accesstoken"
	"github.com/bytom/errors"
)

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()

	var token *string
	tokenDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	accessTokens := accesstoken.NewStore(tokenDB)
	token, err := accessTokens.Create(ctx, "alice", "test")
	if err != nil {
		t.Errorf("create token error")
	}

	cases := []struct {
		id, tok string
		want    error
	}{
		{"alice", *token, nil},
		{"alice", "alice:abcsdsdfassdfsefsfsfesfesfefsefa", ErrInvalidToken},
	}

	api := NewAPI(accessTokens)

	for _, c := range cases {
		var username, password string
		toks := strings.SplitN(c.tok, ":", 2)
		if len(toks) > 0 {
			username = toks[0]
		}
		if len(toks) > 1 {
			password = toks[1]
		}

		req, _ := http.NewRequest("GET", "/", nil)
		req.SetBasicAuth(username, password)

		_, err := api.Authenticate(req)
		if errors.Root(err) != c.want {
			t.Errorf("Authenticate(%s) error = %s want %s", c.id, err, c.want)
		}
	}
}
