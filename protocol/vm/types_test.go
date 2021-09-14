package vm

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/holiman/uint256"

	"github.com/bytom/bytom/testutil"
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

func TestInt64BigIntConvert(t *testing.T) {
	cases := []uint64{0, 1, 2, 1024, 65536, 9223372036854775807}
	for i, c := range cases {
		x := Uint64Bytes(c)
		y := BigIntBytes(uint256.NewInt(c))
		if !testutil.DeepEqual(x, y) {
			t.Errorf("case %d fail on compare %d bytes", i, c)
		}
	}
}

func Test_reverse(t *testing.T) {
	type args struct {
		b []byte
	}
	type wants struct {
		origin []byte
		want   []byte
	}
	tests := []struct {
		name  string
		args  args
		wants wants
	}{
		{
			name: "test reverse",
			args: args{
				b: []byte{0x00, 0x00, 0x00, 0x00, 0x01},
			},
			wants: wants{
				origin: []byte{0x00, 0x00, 0x00, 0x00, 0x01},
				want:   []byte{0x01, 0x00, 0x00, 0x00, 0x00},
			},
		},
		{
			name: "test reverse 1",
			args: args{
				b: []byte{0x01, 0x02, 0x20, 0x03, 0x01},
			},
			wants: wants{
				origin: []byte{0x01, 0x02, 0x20, 0x03, 0x01},
				want:   []byte{0x01, 0x03, 0x20, 0x02, 0x01},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reverse(tt.args.b); !reflect.DeepEqual(got, tt.wants.want) {
				t.Errorf("reverse() = %v, want %v", got, tt.wants.want)
			}
			if !reflect.DeepEqual(tt.args.b, tt.wants.origin) {
				t.Errorf("after reverse args = %v, origin %v", tt.args.b, tt.wants.origin)
			}
		})
	}
}
