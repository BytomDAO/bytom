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
	KeyIndex uint32         `json:"key_index"`
}

// Path returns the complete path for derived keys
// path format /change/index
func Path(change bool, itemIndexes ...uint64) [][]byte {
	var path [][]byte
	changePath := make([]byte, 1)
	if change == true {
		changePath[0] = 1
	} else {
		changePath[0] = 0
	}
	path = append(path, changePath[:])

	for _, idx := range itemIndexes {
		var idxBytes [8]byte
		binary.LittleEndian.PutUint64(idxBytes[:], idx)
		path = append(path, idxBytes[:])
	}
	return path
}

// Create creates and stores a Signer in the database
func Create(ctx context.Context, db dbm.DB, signerType string, xpubs []chainkd.XPub, quorum int, keyIndex uint32, clientToken string) (string, *Signer, error) {
	if len(xpubs) == 0 {
		return "", nil, errors.Wrap(ErrNoXPubs)
	}

	sort.Sort(sortKeys(xpubs)) // this transforms the input slice
	for i := 1; i < len(xpubs); i++ {
		if bytes.Equal(xpubs[i].Bytes(), xpubs[i-1].Bytes()) {
			return "", nil, errors.WithDetailf(ErrDupeXPub, "duplicated key=%x", xpubs[i])
		}
	}

	if quorum == 0 || quorum > len(xpubs) {
		return "", nil, errors.Wrap(ErrBadQuorum)
	}

	var xpubBytes [][]byte
	for _, key := range xpubs {
		key := key
		xpubBytes = append(xpubBytes, key.Bytes())
	}

	id := IDGenerate()
	return id, &Signer{
		Type:     signerType,
		XPubs:    xpubs,
		Quorum:   quorum,
		KeyIndex: keyIndex,
	}, nil
}

type sortKeys []chainkd.XPub

func (s sortKeys) Len() int           { return len(s) }
func (s sortKeys) Less(i, j int) bool { return bytes.Compare(s[i].Bytes(), s[j].Bytes()) < 0 }
func (s sortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
