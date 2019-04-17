package vmutil

import (
	"encoding/hex"
	"testing"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/errors"
)

// TestIsUnspendable ensures the IsUnspendable function returns the expected
// results.
func TestIsUnspendable(t *testing.T) {
	tests := []struct {
		pkScript []byte
		expected bool
	}{
		{
			// Unspendable
			pkScript: []byte{0x6a, 0x04, 0x74, 0x65, 0x73, 0x74},
			expected: true,
		},
		{
			// Spendable
			pkScript: []byte{0x76, 0xa9, 0x14, 0x29, 0x95, 0xa0,
				0xfe, 0x68, 0x43, 0xfa, 0x9b, 0x95, 0x45,
				0x97, 0xf0, 0xdc, 0xa7, 0xa4, 0x4d, 0xf6,
				0xfa, 0x0b, 0x5c, 0x88, 0xac},
			expected: false,
		},
	}

	for i, test := range tests {
		res := IsUnspendable(test.pkScript)
		if res != test.expected {
			t.Errorf("TestIsUnspendable #%d failed: got %v want %v",
				i, res, test.expected)
			continue
		}
	}
}

func TestP2SPMultiSigProgram(t *testing.T) {
	pub1, _ := hex.DecodeString("988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6")
	pub2, _ := hex.DecodeString("7192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c62")
	pub3, _ := hex.DecodeString("8bcd251d9f4e03877130b6e6f1d577eda562375f07c3cdfad8f1d541002fd1a3")

	tests := []struct {
		pubkeys     []ed25519.PublicKey
		nrequired   int
		wantProgram string
		wantErr     error
	}{
		{
			pubkeys:     []ed25519.PublicKey{pub1},
			nrequired:   1,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb65151ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1, pub2},
			nrequired:   2,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6207192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c625252ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1, pub2, pub3},
			nrequired:   2,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6207192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c62208bcd251d9f4e03877130b6e6f1d577eda562375f07c3cdfad8f1d541002fd1a35253ad",
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1},
			nrequired: -1,
			wantErr:   errors.WithDetail(ErrBadValue, "negative quorum"),
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1},
			nrequired: 0,
			wantErr:   errors.WithDetail(ErrBadValue, "quorum empty with non-empty pubkey list"),
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1, pub2},
			nrequired: 3,
			wantErr:   errors.WithDetail(ErrBadValue, "quorum too big"),
		},
	}

	for i, test := range tests {
		got, err := P2SPMultiSigProgram(test.pubkeys, test.nrequired)
		if err != nil {
			if test.wantErr != nil && err.Error() != test.wantErr.Error() {
				t.Errorf("TestP2SPMultiSigProgram #%d failed: got %v want %v", i, err.Error(), test.wantErr.Error())
			} else if test.wantErr == nil {
				t.Fatal(err)
			}
		}

		if hex.EncodeToString(got) != test.wantProgram {
			t.Errorf("TestP2SPMultiSigProgram #%d failed: got %v want %v", i, hex.EncodeToString(got), test.wantProgram)
		}
	}
}

func TestP2SPMultiSigProgramWithHeight(t *testing.T) {
	pub1, _ := hex.DecodeString("988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6")
	pub2, _ := hex.DecodeString("7192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c62")
	pub3, _ := hex.DecodeString("8bcd251d9f4e03877130b6e6f1d577eda562375f07c3cdfad8f1d541002fd1a3")

	tests := []struct {
		pubkeys     []ed25519.PublicKey
		nrequired   int
		height      int64
		wantProgram string
		wantErr     error
	}{
		{
			pubkeys:     []ed25519.PublicKey{pub1},
			nrequired:   1,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb65151ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1, pub2},
			nrequired:   2,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6207192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c625252ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1, pub2, pub3},
			nrequired:   2,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6207192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c62208bcd251d9f4e03877130b6e6f1d577eda562375f07c3cdfad8f1d541002fd1a35253ad",
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1},
			nrequired: 1,
			height:    -1,
			wantErr:   errors.WithDetail(ErrBadValue, "negative blockHeight"),
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1},
			nrequired:   1,
			height:      0,
			wantProgram: "ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb65151ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1},
			nrequired:   1,
			height:      200,
			wantProgram: "01c8cda069ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb65151ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1, pub2},
			nrequired:   2,
			height:      200,
			wantProgram: "01c8cda069ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6207192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c625252ad",
		},
		{
			pubkeys:     []ed25519.PublicKey{pub1, pub2, pub3},
			nrequired:   2,
			height:      200,
			wantProgram: "01c8cda069ae20988650ff921c82d47a953527894f792572ba63197c56e5fe79e5df0c444d6bb6207192bf4eac0789ee19c88dfa87861cf59e215820f7bdb7be02761d9ed92e6c62208bcd251d9f4e03877130b6e6f1d577eda562375f07c3cdfad8f1d541002fd1a35253ad",
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1},
			nrequired: -1,
			wantErr:   errors.WithDetail(ErrBadValue, "negative quorum"),
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1},
			nrequired: 0,
			wantErr:   errors.WithDetail(ErrBadValue, "quorum empty with non-empty pubkey list"),
		},
		{
			pubkeys:   []ed25519.PublicKey{pub1, pub2},
			nrequired: 3,
			wantErr:   errors.WithDetail(ErrBadValue, "quorum too big"),
		},
	}

	for i, test := range tests {
		got, err := P2SPMultiSigProgramWithHeight(test.pubkeys, test.nrequired, test.height)
		if err != nil {
			if test.wantErr != nil && err.Error() != test.wantErr.Error() {
				t.Errorf("TestP2SPMultiSigProgram #%d failed: got %v want %v", i, err.Error(), test.wantErr.Error())
			} else if test.wantErr == nil {
				t.Fatal(err)
			}
		}

		if hex.EncodeToString(got) != test.wantProgram {
			t.Errorf("TestP2SPMultiSigProgram #%d failed: got %v want %v", i, hex.EncodeToString(got), test.wantProgram)
		}
	}
}

func TestGetIssuanceProgramRestrictHeight(t *testing.T) {
	tests := []struct {
		issuanceProgram string
		wantHeight      int64
	}{
		{
			issuanceProgram: "",
			wantHeight:      0,
		},
		{
			issuanceProgram: "ae20ac20f5cdb9ada2ae9836bcfff32126d6b885aa3f73ee111a95d1bf37f3904aca5151ad",
			wantHeight:      0,
		},
		{
			issuanceProgram: "01c8cda069ae20f44dd85be89de08b0f894476ccc7b3eebcf0a288c79504fa7e4c8033f5b7338020c86dc682ce3ecac64e165d9b5f8cca9ee05bd0d4df07adbfd11251ad7e88f1685152ad",
			wantHeight:      200,
		},
	}

	for i, test := range tests {
		program, err := hex.DecodeString(test.issuanceProgram)
		if err != nil {
			t.Fatal(err)
		}

		gotHeight := GetIssuanceProgramRestrictHeight(program)
		if gotHeight != test.wantHeight {
			t.Errorf("TestGetIssuanceProgramRestrictHeight #%d failed: got %d want %d", i, gotHeight, test.wantHeight)
		}
	}
}
