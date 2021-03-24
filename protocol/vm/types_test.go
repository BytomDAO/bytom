package vm

import (
	"bytes"
	"math/big"
	"testing"
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
	cases := []struct {
		num  *big.Int
		data []byte
	}{
		{new(big.Int).SetInt64(0), []byte{48}},
		{new(big.Int).SetInt64(1), []byte{0x31}},
		{new(big.Int).SetInt64(255), []byte{0x32, 0x35, 0x35}},
		{new(big.Int).SetInt64(256), []byte{0x32, 0x35, 0x36}},
		{new(big.Int).SetInt64(1 << 16), []byte{0x36, 0x35, 0x35, 0x33, 0x36}},
		{new(big.Int).SetInt64(-1), []byte{0x2d, 0x31}},
		{new(big.Int).SetInt64(-2), []byte{0x2d, 0x32}},
	}

	for _, c := range cases {
		gotData := BigIntBytes1(c.num)
		if !bytes.Equal(gotData, c.data) {
			t.Errorf("BigIntBytes(%d) = %x want %x", c.num, gotData, c.data)
		}

		gotNum, _ := AsBigInt1(c.data)
		if gotNum.Cmp(c.num) != 0 {
			t.Errorf("AsBigInt(%x) = %d want %d", c.data, gotNum, c.num)
		}
	}

	data := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}
	_, err := AsBigInt1(data)
	want := ErrBadValue
	if err != want {
		t.Errorf("AsInt64(%x) = %v want %v", data, err, want)
	}
}

func TestAsBigInt(t *testing.T) {
	cases := []struct {
		num     *big.Int
		data    []byte
		wantErr error
	}{
		{new(big.Int).SetInt64(0), []byte{48}, nil},
		{new(big.Int).SetInt64(1), []byte{0x31}, nil},
		{new(big.Int).SetInt64(255), []byte{0x32, 0x35, 0x35}, nil},
		{new(big.Int).SetInt64(256), []byte{0x32, 0x35, 0x36}, nil},
		{new(big.Int).SetInt64(1 << 16), []byte{0x36, 0x35, 0x35, 0x33, 0x36}, nil},
		{new(big.Int).SetInt64(-1), []byte{0x2d, 0x31}, nil},
		{new(big.Int).SetInt64(-2), []byte{0x2d, 0x32}, nil},
		{nil, []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}, ErrBadValue},
	}

	for _, c := range cases {
		gotNum, err := AsBigInt1(c.data)
		if err != nil {
			if err != c.wantErr {
				t.Errorf("AsBigInt(%x) = %v want %v", c.data, gotNum, c.num)
			}
			continue
		}

		if gotNum.Cmp(c.num) != 0 {
			t.Errorf("AsBigInt(%x) = %d want %d", c.data, gotNum, c.num)
		}
	}
}
