// Package signers associates signers and their corresponding keys.
package signers

import (
	"bytes"
	"encoding/binary"
	"sort"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/errors"
)

type keySpace byte

const (
	AssetKeySpace   keySpace = 0
	AccountKeySpace keySpace = 1
)

var (
	// ErrBadQuorum is returned by Create when the quorum
	// provided is less than 1 or greater than the number
	// of xpubs provided.
	ErrBadQuorum = errors.New("quorum must be greater than or equal to 1, and must be less than or equal to the length of xpubs")

	// ErrBadXPub is returned by Create when the xpub
	// provided isn't valid.
	ErrBadXPub = errors.New("invalid xpub format")

	// ErrNoXPubs is returned by create when the xpubs
	// slice provided is empty.
	ErrNoXPubs = errors.New("at least one xpub is required")

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
func Create(signerType string, xpubs []chainkd.XPub, quorum int, keyIndex uint64) (*Signer, error) {
	if len(xpubs) == 0 {
		return nil, errors.Wrap(ErrNoXPubs)
	}

	sort.Sort(sortKeys(xpubs)) // this transforms the input slice
	for i := 1; i < len(xpubs); i++ {
		if bytes.Equal(xpubs[i][:], xpubs[i-1][:]) {
			return nil, errors.WithDetailf(ErrDupeXPub, "duplicated key=%x", xpubs[i])
		}
	}

	if quorum == 0 || quorum > len(xpubs) {
		return nil, errors.Wrap(ErrBadQuorum)
	}

	return &Signer{
		Type:     signerType,
		XPubs:    xpubs,
		Quorum:   quorum,
		KeyIndex: keyIndex,
	}, nil
}

type sortKeys []chainkd.XPub

func (s sortKeys) Len() int           { return len(s) }
func (s sortKeys) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s sortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
