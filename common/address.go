package common

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/bytom/bytom/common/bech32"
	"github.com/bytom/bytom/consensus"
)

var (
	// ErrChecksumMismatch describes an error where decoding failed due
	// to a bad checksum.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrUnknownAddressType describes an error where an address can not
	// decoded as a specific address type due to the string encoding
	// begining with an identifier byte unknown to any standard or
	// registered (via chaincfg.Register) network.
	ErrUnknownAddressType = errors.New("unknown address type")

	// ErrAddressCollision describes an error where an address can not
	// be uniquely determined as either a pay-to-pubkey-hash or
	// pay-to-script-hash address since the leading identifier is used for
	// describing both address kinds, but for different networks.  Rather
	// than assuming or defaulting to one or the other, this error is
	// returned and the caller must decide how to decode the address.
	ErrAddressCollision = errors.New("address collision")

	// ErrUnsupportedWitnessVer describes an error where a segwit address being
	// decoded has an unsupported witness version.
	ErrUnsupportedWitnessVer = errors.New("unsupported witness version")

	// ErrUnsupportedWitnessProgLen describes an error where a segwit address
	// being decoded has an unsupported witness program length.
	ErrUnsupportedWitnessProgLen = errors.New("unsupported witness program length")
)

// Address is an interface type for any type of destination a transaction
// output may spend to.  This includes pay-to-pubkey (P2PK), pay-to-pubkey-hash
// (P2PKH), and pay-to-script-hash (P2SH).  Address is designed to be generic
// enough that other kinds of addresses may be added in the future without
// changing the decoding and encoding API.
type Address interface {
	// String returns the string encoding of the transaction output
	// destination.
	//
	// Please note that String differs subtly from EncodeAddress: String
	// will return the value as a string without any conversion, while
	// EncodeAddress may convert destination types (for example,
	// converting pubkeys to P2PKH addresses) before encoding as a
	// payment address string.
	String() string

	// EncodeAddress returns the string encoding of the payment address
	// associated with the Address value.  See the comment on String
	// for how this method differs from String.
	EncodeAddress() string

	// ScriptAddress returns the raw bytes of the address to be used
	// when inserting the address into a txout's script.
	ScriptAddress() []byte

	// IsForNet returns whether or not the address is associated with the
	// passed bytom network.
	IsForNet(*consensus.Params) bool
}

// encodeSegWitAddress creates a bech32 encoded address string representation
// from witness version and witness program.
func encodeSegWitAddress(hrp string, witnessVersion byte, witnessProgram []byte) (string, error) {
	// Group the address bytes into 5 bit groups, as this is what is used to
	// encode each character in the address string.
	converted, err := bech32.ConvertBits(witnessProgram, 8, 5, true)
	if err != nil {
		return "", err
	}

	// Concatenate the witness version and program, and encode the resulting
	// bytes using bech32 encoding.
	combined := make([]byte, len(converted)+1)
	combined[0] = witnessVersion
	copy(combined[1:], converted)
	bech, err := bech32.Bech32Encode(hrp, combined)
	if err != nil {
		return "", err
	}

	// Check validity by decoding the created address.
	version, program, err := decodeSegWitAddress(bech)
	if err != nil {
		return "", fmt.Errorf("invalid segwit address: %v", err)
	}

	if version != witnessVersion || !bytes.Equal(program, witnessProgram) {
		return "", fmt.Errorf("invalid segwit address")
	}

	return bech, nil
}

// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type.
//
// The bytom network the address is associated with is extracted if possible.
// When the address does not encode the network, such as in the case of a raw
// public key, the address will be associated with the passed defaultNet.
func DecodeAddress(addr string, param *consensus.Params) (Address, error) {
	// Bech32 encoded segwit addresses start with a human-readable part
	// (hrp) followed by '1'. For Bytom mainnet the hrp is "bm", and for
	// testnet it is "tm". If the address string has a prefix that matches
	// one of the prefixes for the known networks, we try to decode it as
	// a segwit address.
	oneIndex := strings.LastIndexByte(addr, '1')
	if oneIndex > 1 {
		prefix := addr[:oneIndex+1]
		if consensus.IsBech32SegwitPrefix(prefix, param) {
			witnessVer, witnessProg, err := decodeSegWitAddress(addr)
			if err != nil {
				return nil, err
			}

			// We currently only support P2WPKH and P2WSH, which is
			// witness version 0.
			if witnessVer != 0 {
				return nil, ErrUnsupportedWitnessVer
			}

			// The HRP is everything before the found '1'.
			hrp := prefix[:len(prefix)-1]

			switch len(witnessProg) {
			case 20:
				return newAddressWitnessPubKeyHash(hrp, witnessProg)
			case 32:
				return newAddressWitnessScriptHash(hrp, witnessProg)
			default:
				return nil, ErrUnsupportedWitnessProgLen
			}
		}
	}
	return nil, ErrUnknownAddressType
}

// decodeSegWitAddress parses a bech32 encoded segwit address string and
// returns the witness version and witness program byte representation.
func decodeSegWitAddress(address string) (byte, []byte, error) {
	// Decode the bech32 encoded address.
	_, data, err := bech32.Bech32Decode(address)
	if err != nil {
		return 0, nil, err
	}

	// The first byte of the decoded address is the witness version, it must
	// exist.
	if len(data) < 1 {
		return 0, nil, fmt.Errorf("no witness version")
	}

	// ...and be <= 16.
	version := data[0]
	if version > 16 {
		return 0, nil, fmt.Errorf("invalid witness version: %v", version)
	}

	// The remaining characters of the address returned are grouped into
	// words of 5 bits. In order to restore the original witness program
	// bytes, we'll need to regroup into 8 bit words.
	regrouped, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return 0, nil, err
	}

	// The regrouped data must be between 2 and 40 bytes.
	if len(regrouped) < 2 || len(regrouped) > 40 {
		return 0, nil, fmt.Errorf("invalid data length")
	}

	// For witness version 0, address MUST be exactly 20 or 32 bytes.
	if version == 0 && len(regrouped) != 20 && len(regrouped) != 32 {
		return 0, nil, fmt.Errorf("invalid data length for witness "+
			"version 0: %v", len(regrouped))
	}

	return version, regrouped, nil
}

// AddressWitnessPubKeyHash is an Address for a pay-to-witness-pubkey-hash
// (P2WPKH) output. See BIP 173 for further details regarding native segregated
// witness address encoding:
// https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki
type AddressWitnessPubKeyHash struct {
	hrp            string
	witnessVersion byte
	witnessProgram [20]byte
}

// NewAddressWitnessPubKeyHash returns a new AddressWitnessPubKeyHash.
func NewAddressWitnessPubKeyHash(witnessProg []byte, param *consensus.Params) (*AddressWitnessPubKeyHash, error) {
	return newAddressWitnessPubKeyHash(param.Bech32HRPSegwit, witnessProg)
}

// newAddressWitnessPubKeyHash is an internal helper function to create an
// AddressWitnessPubKeyHash with a known human-readable part, rather than
// looking it up through its parameters.
func newAddressWitnessPubKeyHash(hrp string, witnessProg []byte) (*AddressWitnessPubKeyHash, error) {
	// Check for valid program length for witness version 0, which is 20
	// for P2WPKH.
	if len(witnessProg) != 20 {
		return nil, errors.New("witness program must be 20 bytes for p2wpkh")
	}

	addr := &AddressWitnessPubKeyHash{
		hrp:            strings.ToLower(hrp),
		witnessVersion: 0x00,
	}

	copy(addr.witnessProgram[:], witnessProg)

	return addr, nil
}

// EncodeAddress returns the bech32 string encoding of an
// AddressWitnessPubKeyHash.
// Part of the Address interface.
func (a *AddressWitnessPubKeyHash) EncodeAddress() string {
	str, err := encodeSegWitAddress(a.hrp, a.witnessVersion,
		a.witnessProgram[:])
	if err != nil {
		return ""
	}
	return str
}

// ScriptAddress returns the witness program for this address.
// Part of the Address interface.
func (a *AddressWitnessPubKeyHash) ScriptAddress() []byte {
	return a.witnessProgram[:]
}

// IsForNet returns whether or not the AddressWitnessPubKeyHash is associated
// with the passed bitcoin network.
// Part of the Address interface.
func (a *AddressWitnessPubKeyHash) IsForNet(param *consensus.Params) bool {
	return a.hrp == param.Bech32HRPSegwit
}

// String returns a human-readable string for the AddressWitnessPubKeyHash.
// This is equivalent to calling EncodeAddress, but is provided so the type
// can be used as a fmt.Stringer.
// Part of the Address interface.
func (a *AddressWitnessPubKeyHash) String() string {
	return a.EncodeAddress()
}

// Hrp returns the human-readable part of the bech32 encoded
// AddressWitnessPubKeyHash.
func (a *AddressWitnessPubKeyHash) Hrp() string {
	return a.hrp
}

// WitnessVersion returns the witness version of the AddressWitnessPubKeyHash.
func (a *AddressWitnessPubKeyHash) WitnessVersion() byte {
	return a.witnessVersion
}

// WitnessProgram returns the witness program of the AddressWitnessPubKeyHash.
func (a *AddressWitnessPubKeyHash) WitnessProgram() []byte {
	return a.witnessProgram[:]
}

// Hash160 returns the witness program of the AddressWitnessPubKeyHash as a
// byte array.
func (a *AddressWitnessPubKeyHash) Hash160() *[20]byte {
	return &a.witnessProgram
}

// AddressWitnessScriptHash is an Address for a pay-to-witness-script-hash
// (P2WSH) output. See BIP 173 for further details regarding native segregated
// witness address encoding:
// https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki
type AddressWitnessScriptHash struct {
	hrp            string
	witnessVersion byte
	witnessProgram [32]byte
}

// NewAddressWitnessScriptHash returns a new AddressWitnessPubKeyHash.
func NewAddressWitnessScriptHash(witnessProg []byte, param *consensus.Params) (*AddressWitnessScriptHash, error) {
	return newAddressWitnessScriptHash(param.Bech32HRPSegwit, witnessProg)
}

// newAddressWitnessScriptHash is an internal helper function to create an
// AddressWitnessScriptHash with a known human-readable part, rather than
// looking it up through its parameters.
func newAddressWitnessScriptHash(hrp string, witnessProg []byte) (*AddressWitnessScriptHash, error) {
	// Check for valid program length for witness version 0, which is 32
	// for P2WSH.
	if len(witnessProg) != 32 {
		return nil, errors.New("witness program must be 32 bytes for p2wsh")
	}

	addr := &AddressWitnessScriptHash{
		hrp:            strings.ToLower(hrp),
		witnessVersion: 0x00,
	}

	copy(addr.witnessProgram[:], witnessProg)

	return addr, nil
}

// EncodeAddress returns the bech32 string encoding of an
// AddressWitnessScriptHash.
// Part of the Address interface.
func (a *AddressWitnessScriptHash) EncodeAddress() string {
	str, err := encodeSegWitAddress(a.hrp, a.witnessVersion,
		a.witnessProgram[:])
	if err != nil {
		return ""
	}
	return str
}

// ScriptAddress returns the witness program for this address.
// Part of the Address interface.
func (a *AddressWitnessScriptHash) ScriptAddress() []byte {
	return a.witnessProgram[:]
}

// IsForNet returns whether or not the AddressWitnessScriptHash is associated
// with the passed bytom network.
// Part of the Address interface.
func (a *AddressWitnessScriptHash) IsForNet(param *consensus.Params) bool {
	return a.hrp == param.Bech32HRPSegwit
}

// String returns a human-readable string for the AddressWitnessScriptHash.
// This is equivalent to calling EncodeAddress, but is provided so the type
// can be used as a fmt.Stringer.
// Part of the Address interface.
func (a *AddressWitnessScriptHash) String() string {
	return a.EncodeAddress()
}

// Hrp returns the human-readable part of the bech32 encoded
// AddressWitnessScriptHash.
func (a *AddressWitnessScriptHash) Hrp() string {
	return a.hrp
}

// WitnessVersion returns the witness version of the AddressWitnessScriptHash.
func (a *AddressWitnessScriptHash) WitnessVersion() byte {
	return a.witnessVersion
}

// WitnessProgram returns the witness program of the AddressWitnessScriptHash.
func (a *AddressWitnessScriptHash) WitnessProgram() []byte {
	return a.witnessProgram[:]
}

// Sha256 returns the witness program of the AddressWitnessPubKeyHash as a
// byte array.
func (a *AddressWitnessScriptHash) Sha256() *[32]byte {
	return &a.witnessProgram
}
