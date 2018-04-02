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
				0: false,
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
		for i := 0; i < len(c.op); i++ {
			if err := ts.SetStatus(i, c.op[i]); err != nil {
				t.Errorf("test case #%d, %t", ci, err)
			}
		}

		for i, v := range c.result {
			result, err := ts.GetStatus(i)
			if err != nil {
				t.Errorf("test case #%d, %t", ci, err)
			}
			if result != v {
				t.Errorf("bad result, %d want %t get %t", i, v, result)
			}
		}
	}
}
