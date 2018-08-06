package bc

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/encoding/blockchain"
)

// NewAssetID convert byte array to aseet id
func NewAssetID(b [32]byte) (a AssetID) {
	return AssetID{
		V0: binary.BigEndian.Uint64(b[0:8]),
		V1: binary.BigEndian.Uint64(b[8:16]),
		V2: binary.BigEndian.Uint64(b[16:24]),
		V3: binary.BigEndian.Uint64(b[24:32]),
	}
}

// Byte32 return the byte array representation
func (a AssetID) Byte32() (b32 [32]byte) { return Hash(a).Byte32() }

// MarshalText satisfies the TextMarshaler interface.
func (a AssetID) MarshalText() ([]byte, error) { return Hash(a).MarshalText() }

// UnmarshalText satisfies the TextUnmarshaler interface.
func (a *AssetID) UnmarshalText(b []byte) error { return (*Hash)(a).UnmarshalText(b) }

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (a *AssetID) UnmarshalJSON(b []byte) error { return (*Hash)(a).UnmarshalJSON(b) }

// Bytes returns the byte representation.
func (a AssetID) Bytes() []byte { return Hash(a).Bytes() }

// WriteTo satisfies the io.WriterTo interface.
func (a AssetID) WriteTo(w io.Writer) (int64, error) { return Hash(a).WriteTo(w) }

// ReadFrom satisfies the io.ReaderFrom interface.
func (a *AssetID) ReadFrom(r io.Reader) (int64, error) { return (*Hash)(a).ReadFrom(r) }

// IsZero tells whether a Asset pointer is nil or points to an all-zero hash.
func (a *AssetID) IsZero() bool { return (*Hash)(a).IsZero() }

// ComputeAssetID calculate the asset id from AssetDefinition
func (ad *AssetDefinition) ComputeAssetID() (assetID AssetID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	writeForHash(h, *ad) // error is impossible
	var b [32]byte
	h.Read(b[:]) // error is impossible
	return NewAssetID(b)
}

// ComputeAssetID implement the assetID calculate logic
func ComputeAssetID(prog []byte, vmVersion uint64, data *Hash) AssetID {
	def := &AssetDefinition{
		IssuanceProgram: &Program{
			VmVersion: vmVersion,
			Code:      prog,
		},
		Data: data,
	}
	return def.ComputeAssetID()
}

// ReadFrom read the AssetAmount from the bytes
func (a *AssetAmount) ReadFrom(r *blockchain.Reader) (err error) {
	var assetID AssetID
	if _, err = assetID.ReadFrom(r); err != nil {
		return err
	}
	a.AssetId = &assetID
	a.Amount, err = blockchain.ReadVarint63(r)
	return err
}

// WriteTo convert struct to byte and write to io
func (a AssetAmount) WriteTo(w io.Writer) (int64, error) {
	n, err := a.AssetId.WriteTo(w)
	if err != nil {
		return n, err
	}
	n2, err := blockchain.WriteVarint63(w, a.Amount)
	return n + int64(n2), err
}

// Equal check does two AssetAmount have same assetID and amount
func (a *AssetAmount) Equal(other *AssetAmount) (eq bool, err error) {
	if a == nil || other == nil {
		return false, errors.New("empty asset amount")
	}
	if a.AssetId == nil || other.AssetId == nil {
		return false, errors.New("empty asset id")
	}
	return a.Amount == other.Amount && *a.AssetId == *other.AssetId, nil
}
