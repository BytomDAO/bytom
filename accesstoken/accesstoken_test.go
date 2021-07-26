package accesstoken

import (
	"context"
	"os"
	"strings"
	"testing"

	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/errors"
)

func TestCreate(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	cases := []struct {
		id, typ string
		want    error
	}{
		{"a", "client", nil},
		{"b", "network", nil},
		{"", "client", ErrBadID},
		{"bad:id", "client", ErrBadID},
		{"a", "network", ErrDuplicateID}, // this aborts the transaction, so no tests can follow
	}

	for _, c := range cases {
		_, err := cs.Create(c.id, c.typ)
		if errors.Root(err) != c.want {
			t.Errorf("Create(%s, %s) error = %s want %s", c.id, c.typ, err, c.want)
		}
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	tokenMap := make(map[string]*Token)
	tokenMap["ab"] = mustCreateToken(ctx, t, cs, "ab", "test")
	tokenMap["bc"] = mustCreateToken(ctx, t, cs, "bc", "test")
	tokenMap["cd"] = mustCreateToken(ctx, t, cs, "cd", "test")

	got, err := cs.List()
	if err != nil {
		t.Errorf("List errored: get list error")
	}

	if len(got) != len(tokenMap) {
		t.Error("List errored: get invalid length")
	}
	for _, v := range got {
		if v.Token != tokenMap[v.ID].Token {
			t.Errorf("List error: ID: %s, expected token: %s, DB token: %s", v.ID, *tokenMap[v.ID], v.Token)
		}
		continue
	}
}

func TestCheck(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	token := mustCreateToken(ctx, t, cs, "x", "client")
	tokenParts := strings.Split(token.Token, ":")

	if err := cs.Check(tokenParts[0], tokenParts[1]); err != nil {
		t.Fatal(err)
	}

	if err := cs.Check("x", "badsecret"); err != ErrInvalidToken {
		t.Fatal("invalid token check passed")
	}
}

func mustCreateToken(ctx context.Context, t *testing.T, cs *CredentialStore, id, typ string) *Token {
	token, err := cs.Create(id, typ)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
