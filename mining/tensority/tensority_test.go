package tensority

import (
	"reflect"
	"testing"

	"github.com/bytom/common/hexutil"
	"github.com/bytom/protocol/bc"
)

// Tests that tensority hash result is correct.
func TestHash(t *testing.T) {
	tests := []struct {
		blockHeader []byte
		seed        []byte
		hash        []byte
	}{
		{
			blockHeader: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			seed:        hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			hash:        hexutil.MustDecode("0xc5d4a19ce842fee409696d14e483f9efe4a7ecc036d1cfeba0199f13f14dc90f"),
		},
		{
			blockHeader: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			seed:        hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			hash:        hexutil.MustDecode("0xc5d4a19ce842fee409696d14e483f9efe4a7ecc036d1cfeba0199f13f14dc90f"),
		},
		{
			blockHeader: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			seed:        hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			hash:        hexutil.MustDecode("0xc5d4a19ce842fee409696d14e483f9efe4a7ecc036d1cfeba0199f13f14dc90f"),
		},
	}
	for i, tt := range tests {
		var b32hash [32]byte
		var b32seed [32]byte

		copy(b32hash[:], tt.blockHeader)
		copy(b32seed[:], tt.seed)
		bhhash := bc.NewHash(b32hash)
		bhseed := bc.NewHash(b32seed)

		result := Hash(&bhhash, &bhseed).Bytes()

		if !reflect.DeepEqual(result, tt.hash) {
			t.Errorf("hash %d: content mismatch: have %x, correct %x", i, result, tt.hash)
		}
	}
}
