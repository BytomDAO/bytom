// Package accesstoken provides storage and validation of Chain Core
// credentials.
package accesstoken

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
)

const tokenSize = 32

var (
	// ErrBadID is returned when Create is called on an invalid id string.
	ErrBadID = errors.New("invalid id")
	// ErrDuplicateID is returned when Create is called on an existing ID.
	ErrDuplicateID = errors.New("duplicate access token ID")
	// ErrBadType is returned when Create is called with a bad type.
	ErrBadType = errors.New("type must be client or network")
	// ErrNoMatchID is returned when Delete is called on nonexisting ID.
	ErrNoMatchID = errors.New("nonexisting access token ID")

	// validIDRegexp checks that all characters are alphumeric, _ or -.
	// It also must have a length of at least 1.
	validIDRegexp = regexp.MustCompile(`^[\w-]+$`)
)

// Token describe the access token.
type Token struct {
	ID      string    `json:"id"`
	Token   string    `json:"token,omitempty"`
	Type    string    `json:"type,omitempty"`
	Created time.Time `json:"created_at"`
}

// CredentialStore store user access credential.
type CredentialStore struct {
	DB dbm.DB
}

// NewStore creates and returns a new Store object.
func NewStore(db dbm.DB) *CredentialStore {
	return &CredentialStore{
		DB: db,
	}
}

// Create generates a new access token with the given ID.
func (cs *CredentialStore) Create(ctx context.Context, id string, typ string) (*string, error) {
	if !validIDRegexp.MatchString(id) {
		return nil, errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}

	key := []byte(id)
	if cs.DB.Get(key) != nil {
		return nil, errors.WithDetailf(ErrDuplicateID, "id %q already in use", id)
	}

	secret := make([]byte, tokenSize)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, err
	}

	hashedSecret := make([]byte, tokenSize)
	sha3pool.Sum256(hashedSecret, secret)
	created := time.Now()

	token := &Token{
		ID:      id,
		Token:   fmt.Sprintf("%s:%x", id, hashedSecret),
		Type:    typ,
		Created: created,
	}

	value, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	cs.DB.Set(key, value)
	hexsec := fmt.Sprintf("%s:%x", id, secret)
	return &hexsec, nil
}

// Check returns whether or not an id-secret pair is a valid access token.
func (cs *CredentialStore) Check(ctx context.Context, id string, secret []byte) (bool, error) {
	if !validIDRegexp.MatchString(id) {
		return false, errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}

	var toHash [tokenSize]byte
	var hashed [tokenSize]byte
	copy(toHash[:], secret)
	sha3pool.Sum256(hashed[:], toHash[:])
	inToken := fmt.Sprintf("%s:%x", id, hashed[:])

	var value []byte
	token := &Token{}

	key := []byte(id)
	if value = cs.DB.Get(key); value == nil {
		return false, errors.WithDetailf(ErrNoMatchID, "check id %q nonexisting", id)
	}
	if err := json.Unmarshal(value, token); err != nil {
		return false, err
	}

	if strings.Compare(token.Token, inToken) == 0 {
		return true, nil
	}

	return false, nil
}

// List lists all access tokens.
func (cs *CredentialStore) List(ctx context.Context) ([]*Token, error) {
	tokens := make([]*Token, 0)
	iter := cs.DB.Iterator()
	defer iter.Release()

	for iter.Next() {
		token := &Token{}
		if err := json.Unmarshal(iter.Value(), token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// Delete deletes an access token by id.
func (cs *CredentialStore) Delete(ctx context.Context, id string) error {
	if !validIDRegexp.MatchString(id) {
		return errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}

	cs.DB.Delete([]byte(id))
	return nil
}
