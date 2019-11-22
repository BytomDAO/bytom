// Package signers associates signers and their corresponding keys.
package signers

import (
	"bytes"
	"encoding/binary"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/errors"
)

type keySpace byte

const (
	AssetKeySpace   keySpace = 0
	AccountKeySpace keySpace = 1
)

const (
	//BIP0032 compatible previous derivation rule m/account/address_index
	BIP0032 uint8 = iota
	//BIP0032 path derivation rule m/purpose'/coin_type'/account'/change/address_index
	BIP0044
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
	ErrDupeXPub   = errors.New("xpubs cannot contain the same key more than once")
	ErrDeriveRule = errors.New("invalid key derive rule")
)

var (
	// BIP44Purpose purpose field 0x0000002c little-endian mode.
	BIP44Purpose = []byte{0x2C, 0x00, 0x00, 0x00}
	// BTMCoinType coin type field 0x00000099 little-endian mode.
	BTMCoinType = []byte{0x99, 0x00, 0x00, 0x00}
)

// Signer is the abstract concept of a signer,
// which is composed of a set of keys as well as
// the amount of signatures needed for quorum.
type Signer struct {
	Type       string         `json:"type"`
	XPubs      []chainkd.XPub `json:"xpubs"`
	Quorum     int            `json:"quorum"`
	KeyIndex   uint64         `json:"key_index"`
	DeriveRule uint8          `json:"derive_rule"`
}

// GetBip0032Path returns the complete path for bip0032 derived keys
func GetBip0032Path(s *Signer, ks keySpace, itemIndexes ...uint64) [][]byte {
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

// getBip0044Path returns the complete path for bip0044 derived keys
func getBip0044Path(accountIndex uint64, change bool, addrIndex uint64) [][]byte {
	var path [][]byte
	path = append(path, BIP44Purpose[:]) //purpose
	path = append(path, BTMCoinType[:])  //coin type
	accIdxBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(accIdxBytes, uint32(accountIndex))
	path = append(path, accIdxBytes) //account index
	branchBytes := make([]byte, 4)
	if change {
		binary.LittleEndian.PutUint32(branchBytes, uint32(1))
	} else {
		binary.LittleEndian.PutUint32(branchBytes, uint32(0))
	}
	path = append(path, branchBytes) //change
	addrIdxBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(addrIdxBytes[:], uint32(addrIndex))
	path = append(path, addrIdxBytes[:]) //address index
	return path
}

// Path returns the complete path for derived keys
func Path(s *Signer, ks keySpace, change bool, addrIndex uint64) ([][]byte, error) {
	switch s.DeriveRule {
	case BIP0032:
		return GetBip0032Path(s, ks, addrIndex), nil
	case BIP0044:
		return getBip0044Path(s.KeyIndex, change, addrIndex), nil
	}
	return nil, ErrDeriveRule
}

// Create creates and stores a Signer in the database
func Create(signerType string, xpubs []chainkd.XPub, quorum int, keyIndex uint64, deriveRule uint8) (*Signer, error) {
	if len(xpubs) == 0 {
		return nil, errors.Wrap(ErrNoXPubs)
	}

	xpubsMap := map[chainkd.XPub]bool{}
	for _, xpub := range xpubs {
		if _, ok := xpubsMap[xpub]; ok {
			return nil, errors.WithDetailf(ErrDupeXPub, "duplicated key=%x", xpub)
		}
		xpubsMap[xpub] = true
	}

	if quorum == 0 || quorum > len(xpubs) {
		return nil, errors.Wrap(ErrBadQuorum)
	}

	return &Signer{
		Type:       signerType,
		XPubs:      xpubs,
		Quorum:     quorum,
		KeyIndex:   keyIndex,
		DeriveRule: deriveRule,
	}, nil
}

type SortKeys []chainkd.XPub

func (s SortKeys) Len() int           { return len(s) }
func (s SortKeys) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s SortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
