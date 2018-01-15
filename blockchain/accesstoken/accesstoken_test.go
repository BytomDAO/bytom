package accesstoken

import (
	"context"
	"os"
	"strings"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
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
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	token := mustCreateToken(ctx, t, cs, "x", "client")
	tokenParts := strings.Split(*token, ":")

	valid, err := cs.Check(ctx, tokenParts[0], tokenParts[1])
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected token and secret to be valid")
	}

	valid, err = cs.Check(ctx, "x", "badsecret")
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

	const id = "Y"
	mustCreateToken(ctx, t, cs, id, "client")

	err := cs.Delete(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	value := cs.DB.Get([]byte(id))
	if len(value) > 0 {
		t.Fatal("delete fail")
	}
}

func TestDeleteWithInvalidId(t *testing.T) {
	ctx := context.Background()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	err := cs.Delete(ctx, "@")
	if errors.Root(err) != ErrBadID {
		t.Errorf("Deletion with invalid id success, while it should not")
	}
}

func mustCreateToken(ctx context.Context, t *testing.T, cs *CredentialStore, id, typ string) *string {
	token, err := cs.Create(ctx, id, typ)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
