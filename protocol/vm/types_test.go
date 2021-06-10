package vm

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

func TestBoolBytes(t *testing.T) {
	got := BoolBytes(true)
	want := []byte{1}
	if !bytes.Equal(got, want) {
		t.Errorf("BoolBytes(t) = %x want %x", got, want)
	}

	got = BoolBytes(false)
	want = []byte{}
	if !bytes.Equal(got, want) {
		t.Errorf("BoolBytes(f) = %x want %x", got, want)
	}
}

func TestAsBool(t *testing.T) {
	cases := []struct {
		data []byte
		want bool
	}{
		{[]byte{0, 0, 0, 0}, false},
		{[]byte{0}, false},
		{[]byte{}, false},
		{[]byte{1}, true},
		{[]byte{1, 1, 1, 1}, true},
		{[]byte{0, 0, 0, 1}, true},
		{[]byte{1, 0, 0, 0}, true},
		{[]byte{2}, true},
	}

	for _, c := range cases {
		got := AsBool(c.data)

		if got != c.want {
			t.Errorf("AsBool(%x) = %v want %v", c.data, got, c.want)
		}
	}
}

func TestInt64(t *testing.T) {
	cases := []struct {
		num  int64
		data []byte
	}{
		{0, []byte{}},
		{1, []byte{0x01}},
		{255, []byte{0xff}},
		{256, []byte{0x00, 0x01}},
		{1 << 16, []byte{0x00, 0x00, 0x01}},
		{-1, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{-2, []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, c := range cases {
		gotData := Int64Bytes(c.num)

		if !bytes.Equal(gotData, c.data) {
			t.Errorf("Int64Bytes(%d) = %x want %x", c.num, gotData, c.data)
		}

		gotNum, _ := AsInt64(c.data)

		if gotNum != c.num {
			t.Errorf("AsInt64(%x) = %d want %d", c.data, gotNum, c.num)
		}
	}

	data := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}
	_, err := AsInt64(data)
	want := ErrBadValue
	if err != want {
		t.Errorf("AsInt64(%x) = %v want %v", data, err, want)
	}
}

func TestBigIntBytes(t *testing.T) {
	tests := []struct {
		input []byte
		num   *big.Int
	}{
		{num: new(big.Int), input: []byte{}},
		{num: new(big.Int).SetInt64(0), input: []byte{}},
		{num: new(big.Int).SetInt64(1), input: []byte{0x01}},
		{num: new(big.Int).SetInt64(255), input: []byte{0xff}},
		{num: new(big.Int).SetInt64(256), input: []byte{0x00, 0x01}},
		{num: new(big.Int).SetInt64(46657), input: []byte{0x41, 0xb6}},
		{num: new(big.Int).SetInt64(1 << 32), input: []byte{0x00, 0x00, 0x00, 0x00, 0x01}},
		{
			num:   new(big.Int).Exp(new(big.Int).SetInt64(10), new(big.Int).SetInt64(32), nil),
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x81, 0xef, 0xac, 0x85, 0x5b, 0x41, 0x6d, 0x2d, 0xee, 0x04},
		},
		{num: new(big.Int).SetInt64(-1), input: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff}},
		{num: new(big.Int).SetInt64(-256), input: []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff}},
	}
	for _, test := range tests {
		fromBig, b := uint256.FromBig(test.num)
		if b {
			t.Errorf("FromBig overflow")
		}

		gotData := BigIntBytes(fromBig)
		if !bytes.Equal(gotData, test.input) {
			t.Errorf("BigIntBytes(%s) = %x want %x", test.num.String(), gotData, test.input)
		}
	}
}

func TestAsBigInt(t *testing.T) {
	tests := []struct {
		input     []byte
		num       *big.Int
		wantError bool
	}{
		{num: new(big.Int), input: []byte{}},
		{num: new(big.Int), input: []byte{0x00}},
		{num: new(big.Int).SetInt64(0), input: []byte{0x00}},
		{num: new(big.Int).SetInt64(1), input: []byte{0x01}},
		{num: new(big.Int).SetInt64(255), input: []byte{0xff}},
		{num: new(big.Int).SetInt64(256), input: []byte{0x00, 0x01}},
		{num: new(big.Int).SetInt64(46657), input: []byte{0x41, 0xb6}},
		{num: new(big.Int).SetInt64(1 << 32), input: []byte{0x00, 0x00, 0x00, 0x00, 0x01}},
		{
			num:   new(big.Int).Exp(new(big.Int).SetInt64(10), new(big.Int).SetInt64(32), nil),
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x81, 0xef, 0xac, 0x85, 0x5b, 0x41, 0x6d, 0x2d, 0xee, 0x04},
		},
		{num: new(big.Int).SetInt64(-1), input: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, wantError: true},
		{num: new(big.Int).SetInt64(-256), input: []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}, wantError: true},
		{input: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, wantError: true},
	}
	for _, test := range tests {
		data, err := AsBigInt(test.input)
		if err != nil {
			if test.wantError {
				continue
			}
			t.Errorf("AsBigInt(%q) --> err %s", test.input, err.Error())
		}

		fromBig, b := uint256.FromBig(test.num)
		if b {
			t.Errorf("FromBig overflow")
		}

		if data != nil && !data.Eq(fromBig) {
			t.Errorf("AsBigInt(%s) = %x want %x", test.num.String(), data, test.input)
		}
	}
}
