package accesstoken

import (
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
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	tokenMap := make(map[string]*Token)
	tokenMap["ab"] = mustCreateToken(t, cs, "ab", "test")
	tokenMap["bc"] = mustCreateToken(t, cs, "bc", "test")
	tokenMap["cd"] = mustCreateToken(t, cs, "cd", "test")

	got, err := cs.List()
	if err != nil {
		t.Errorf("List errored: get list error")
	}

	if len(got) != len(tokenMap) {
		t.Error("List errored: get invalid length")
	}
	for _, v := range got {
		if m := strings.Compare(v.Token, tokenMap[v.ID].Token); m != 0 {
			t.Errorf("List error: ID: %s, expected token: %s, DB token: %s", v.ID, *tokenMap[v.ID], v.Token)
		}
		continue
	}
}

func TestCheck(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	token := mustCreateToken(t, cs, "x", "client")
	tokenParts := strings.Split(token.Token, ":")

	valid, err := cs.Check(tokenParts[0], tokenParts[1])
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected token and secret to be valid")
	}

	valid, err = cs.Check("x", "badsecret")
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("expected bad secret to not be valid")
	}
}

func TestDelete(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	const id = "Y"
	mustCreateToken(t, cs, id, "client")

	err := cs.Delete(id)
	if err != nil {
		t.Fatal(err)
	}

	value := cs.DB.Get([]byte(id))
	if len(value) > 0 {
		t.Fatal("delete fail")
	}
}

func TestDeleteWithInvalidId(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	cs := NewStore(testDB)

	err := cs.Delete("@")
	if errors.Root(err) != ErrBadID {
		t.Errorf("Deletion with invalid id success, while it should not")
	}
}

func mustCreateToken(t *testing.T, cs *CredentialStore, id, typ string) *Token {
	token, err := cs.Create(id, typ)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
