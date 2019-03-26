package wallet

import (
	"testing"
)

func TestDecodeFormatKey(t *testing.T) {
	cases := []struct {
		BlockHeight uint64
		Position    uint32
	}{
		{
			BlockHeight: 16,
			Position:    16,
		},
	}

	for i, c := range cases {
		formatKey := formatKey(c.BlockHeight, c.Position)
		blockHeight, position, err := decodeFormatKey(formatKey)
		if err != nil {
			t.Errorf("case#%d: decodeFormatKey fail", i)
		}

		if blockHeight != c.BlockHeight {
			t.Errorf("case#%d: decodeFormatKey blockHeight mismatch", i)
		}

		if position != int(c.Position) {
			t.Errorf("case#%d: decodeFormatKey position mismatch", i)
		}
	}
}
