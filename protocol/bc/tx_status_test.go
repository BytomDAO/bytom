package bc

import (
	"testing"
)

func TestSetBits(t *testing.T) {
	cases := []struct {
		op     map[int]bool
		result []bool
	}{
		{
			op: map[int]bool{
				0: true,
			},
			result: []bool{true},
		},
		{
			op: map[int]bool{
				0: false,
			},
			result: []bool{false},
		},
		{
			op: map[int]bool{
				1: true,
			},
			result: []bool{false, true},
		},
		{
			op: map[int]bool{
				0: true,
				1: false,
			},
			result: []bool{true, false},
		},
		{
			op: map[int]bool{
				7: true,
			},
			result: []bool{false, false, false, false, false, false, false, true},
		},
		{
			op: map[int]bool{
				7: false,
			},
			result: []bool{false, false, false, false, false, false, false, false},
		},
		{
			op: map[int]bool{
				8: true,
			},
			result: []bool{false, false, false, false, false, false, false, false, true},
		},
		{
			op: map[int]bool{
				8: false,
			},
			result: []bool{false, false, false, false, false, false, false, false, false},
		},
		{
			op: map[int]bool{
				0: true,
				1: false,
				2: false,
				3: true,
				4: true,
				5: true,
				6: false,
				7: true,
				8: false,
				9: true,
			},
			result: []bool{true, false, false, true, true, true, false, true, false, true},
		},
	}

	for ci, c := range cases {
		ts := NewTransactionStatus()
		for k, v := range c.op {
			if err := ts.SetStatus(k, v); err != nil {
				t.Error(err)
			}
		}

		for i, v := range c.result {
			result, err := ts.GetStatus(i)
			if err != nil {
				t.Error(err)
			}
			if result != v {
				t.Errorf("bad result, %d want %t get %t", i, v, result)
			}
		}
		if len(ts.Bitmap) != (len(c.result)+7)/bitsPerByte {
			t.Errorf("wrong bitmap size, %d want %d get %d", ci, len(c.result)/bitsPerByte+1, len(ts.Bitmap))
		}
	}
}
