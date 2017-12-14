package accesstoken

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/bytom/errors"
	dbm "github.com/tendermint/tmlibs/db"
)

func TestCreate(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
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

func mergeSlice(s1 []string, s2 []string) []string {
	slice := make([]string, len(s1)+len(s2))
	copy(slice, s1)
	copy(slice[len(s1):], s2)
	return slice
}

func TestList(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	mustCreateToken(ctx, t, cs, "ab", "test")
	mustCreateToken(ctx, t, cs, "bc", "test")
	mustCreateToken(ctx, t, cs, "cd", "test")

	cases := struct {
		want []string
	}{
		want: []string{"ab", "bc", "cd"},
	}

	got := make([]string, 0)
	after := ""
	for {
		gotonce, tmpAfter, tmpLast, err := cs.List(after, defGenericPageSize, defGenericPageSize)
		if err != nil {
			t.Errorf("Failed get tokens")
		}
		got = mergeSlice(got, gotonce)
		after = tmpAfter
		if tmpLast {
			break
		}
	}

	token := Token{}
	for i, v := range got {
		if err := json.Unmarshal([]byte(v), &token); err != nil {
			t.Errorf("Failed unmarshal token")
		}
		if m := strings.Compare(token.ID, cases.want[i]); m != 0 {
			t.Errorf("List errored: %s %s", token.ID, cases.want[i])
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
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
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
