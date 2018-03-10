// Package signers associates signers and their corresponding keys.
package signers

import (
	"bytes"
	"context"
	"encoding/binary"
	"sort"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/errors"
	dbm "github.com/tendermint/tmlibs/db"
)

type keySpace byte

const (
	//AssetKeySpace means asset key path type
	AssetKeySpace keySpace = 0
	//AccountKeySpace means account key path type
	AccountKeySpace keySpace = 1
)

var (
	// ErrBadQuorum is returned by Create when the quorum
	// provided is less than 1 or greater than the number
	// of xpubs provided.
	ErrBadQuorum = errors.New("quorum must be greater than 1 and less than or equal to the length of xpubs")

	// ErrBadXPub is returned by Create when the xpub
	// provided isn't valid.
	ErrBadXPub = errors.New("invalid xpub format")

	// ErrNoXPubs is returned by create when the xpubs
	// slice provided is empty.
	ErrNoXPubs = errors.New("at least one xpub is required")

	// ErrBadType is returned when a find operation
	// retrieves a signer that is not the expected type.
	ErrBadType = errors.New("retrieved type does not match expected type")

	// ErrDupeXPub is returned by create when the same xpub
	// appears twice in a single call.
	ErrDupeXPub = errors.New("xpubs cannot contain the same key more than once")
)

// Signer is the abstract concept of a signer,
// which is composed of a set of keys as well as
// the amount of signatures needed for quorum.
type Signer struct {
	Type     string         `json:"type"`
	XPubs    []chainkd.XPub `json:"xpubs"`
	Quorum   int            `json:"quorum"`
	KeyIndex uint64         `json:"key_index"`
}

// Path returns the complete path for derived keys
func Path(s *Signer, ks keySpace, itemIndexes ...uint64) [][]byte {
	var path [][]byte
	signerPath := [9]byte{byte(ks)}
	binary.LittleEndian.PutUint64(signerPath[1:], s.KeyIndex)
	path = append(path, signerPath[:])
	for _, idx := range itemIndexes {
		var idxBytes [8]byte
		binary.LittleEndian.PutUint64(idxBytes[:], idx)
		path = append(path, idxBytes[:])
	}
	return path
}

// Create creates and stores a Signer in the database
func Create(signerType string, xpubs []chainkd.XPub, quorum int, keyIndex uint64) (string, *Signer, error) {
	if len(xpubs) == 0 {
		return "", nil, errors.Wrap(ErrNoXPubs)
	}

	sort.Sort(sortKeys(xpubs)) // this transforms the input slice
	for i := 1; i < len(xpubs); i++ {
		if bytes.Equal(xpubs[i][:], xpubs[i-1][:]) {
			return "", nil, errors.WithDetailf(ErrDupeXPub, "duplicated key=%x", xpubs[i])
		}
	}

	if quorum == 0 || quorum > len(xpubs) {
		return "", nil, errors.Wrap(ErrBadQuorum)
	}

	id := IDGenerate()
	return id, &Signer{
		Type:     signerType,
		XPubs:    xpubs,
		Quorum:   quorum,
		KeyIndex: keyIndex,
	}, nil
}

// Find retrieves a Signer from the database
// using the type and id.
func Find(ctx context.Context, db dbm.DB, typ, id string) (*Signer, error) {
	/*const q = `
		SELECT id, type, xpubs, quorum, key_index
		FROM signers WHERE id=$1
	`
	*/

	var (
		s         Signer
		xpubBytes [][]byte
	)
	/*
		err := db.QueryRowContext(ctx, q, id).Scan(
			&s.ID,
			&s.Type,
			(*pq.ByteaArray)(&xpubBytes),
			&s.Quorum,
			&s.KeyIndex,
		)
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(pg.ErrUserInputNotFound)
		}
		if err != nil {
			return nil, errors.Wrap(err)
		}

		if s.Type != typ {
			return nil, errors.Wrap(ErrBadType)
		}*/

	keys, err := ConvertKeys(xpubBytes)
	if err != nil {
		return nil, errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
	}

	s.XPubs = keys

	return &s, nil
}

//ConvertKeys convert []byte to XPub
func ConvertKeys(xpubs [][]byte) ([]chainkd.XPub, error) {
	var xkeys []chainkd.XPub
	for i, xpub := range xpubs {
		var xkey chainkd.XPub
		if len(xpub) != len(xkey) {
			return nil, errors.WithDetailf(ErrBadXPub, "key %d: xpub is not valid", i)
		}
		copy(xkey[:], xpub)
		xkeys = append(xkeys, xkey)
	}
	return xkeys, nil
}

type sortKeys []chainkd.XPub

func (s sortKeys) Len() int           { return len(s) }
func (s sortKeys) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s sortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
