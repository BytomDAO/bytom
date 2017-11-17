package accesstoken

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
)

func TestCreate(t *testing.T) {
	testDB := dbm.NewDB("testdb1", "leveldb", ".data")
	cs := NewStore(testDB)
	ctx := context.Background()

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
		_, err := cs.Create(ctx, c.id, c.typ)
		if errors.Root(err) != c.want {
			t.Errorf("Create(%s, %s) error = %s want %s", c.id, c.typ, err, c.want)
		}
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb2", "leveldb", ".data")
	cs := NewStore(testDB)

	mustCreateToken(ctx, t, cs, "ab", "test")
	mustCreateToken(ctx, t, cs, "bc", "test")
	mustCreateToken(ctx, t, cs, "cd", "test")

	cases := struct {
		want []string
	}{
		want: []string{"ab", "bc", "cd"},
	}

	got, err := cs.List(ctx)
	if err != nil {
		t.Errorf("List errored: get list error")
	}
	for i, v := range got {
		if m := strings.Compare(v.ID, cases.want[i]); m != 0 {
			t.Errorf("List errored: %s %s", v.ID, cases.want[i])
		}
		continue
	}
}

func TestCheck(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb3", "leveldb", ".data")
	cs := NewStore(testDB)

	token := mustCreateToken(ctx, t, cs, "x", "client")
	tokenParts := strings.Split(*token, ":")
	tokenID := tokenParts[0]
	tokenSecret, err := hex.DecodeString(tokenParts[1])
	if err != nil {
		t.Fatal("bad token secret")
	}

	valid, err := cs.Check(ctx, tokenID, tokenSecret)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected token and secret to be valid")
	}

	valid, err = cs.Check(ctx, "x", []byte("badsecret"))
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("expected bad secret to not be valid")
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb4", "leveldb", ".data")
	cs := NewStore(testDB)

	token := mustCreateToken(ctx, t, cs, "Y", "client")
	tokenParts := strings.Split(*token, ":")
	tokenID := tokenParts[0]

	err := cs.Delete(ctx, tokenID)
	if err != nil {
		t.Fatal(err)
	}
}

func mustCreateToken(ctx context.Context, t *testing.T, cs *CredentialStore, id, typ string) *string {
	token, err := cs.Create(ctx, id, typ)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
