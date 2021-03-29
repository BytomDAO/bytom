// uint256: Fixed size 256-bit math library
// Copyright 2018-2020 uint256 Authors
// SPDX-License-Identifier: BSD-3-Clause

package uint256

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
)

var (
	bigtt256 = new(big.Int).Lsh(big.NewInt(1), 256)
	bigtt255 = new(big.Int).Lsh(big.NewInt(1), 255)

	unTestCases = []string{
		"0",
		"1",
		"0x12cbafcee8f60f9f3fa308c90fde8d298772ffea667aa6bc109d5c661e7929a5",
		"0x8000000000000000000000000000000000000000000000000000000000000000",
		"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
		"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}

	// A collection of interesting input values for binary operators (especially for division).
	// No expected results as big.Int can be used as the source of truth.
	binTestCases = [][2]string{
		{"0", "0"},
		{"1", "0"},
		{"1", "0x767676767676767676000000767676767676"},
		{"2", "0"},
		{"2", "1"},
		{"0x12cbafcee8f60f9f3fa308c90fde8d298772ffea667aa6bc109d5c661e7929a5", "0x00000c76f4afb041407a8ea478d65024f5c3dfe1db1a1bb10c5ea8bec314ccf9"},
		{"0x10000000000000000", "2"},
		{"0x7000000000000000", "0x8000000000000000"},
		{"0x8000000000000000", "0x8000000000000000"},
		{"0x8000000000000001", "0x8000000000000000"},
		{"0x80000000000000010000000000000000", "0x80000000000000000000000000000000"},
		{"0x80000000000000000000000000000000", "0x80000000000000000000000000000001"},
		{"0x478392145435897052", "0x111"},
		{"0x767676767676767676000000767676767676", "0x2900760076761e00020076760000000076767676000000"},
		{"0x12121212121212121212121212121212", "0x232323232323232323"},
		{"0xfffff716b61616160b0b0b2b0b0b0becf4bef50a0df4f48b090b2b0bc60a0a00", "0xfffff716b61616160b0b0b2b0b230b000008010d0a2b00"},
		{"0x50beb1c60141a0000dc2b0b0b0b0b0b410a0a0df4f40b090b2b0bc60a0a00", "0x2000110000000d0a300e750a000000090a0a"},
		{"0x4b00000b41000b0b0b2b0b0b0b0b0b410a0aeff4f40b090b2b0bc60a0a1000", "0x4b00000b41000b0b0b2b0b0b0b0b0b410a0aeff4f40b0a0a"},
		{"0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "7"},
		{"0xf6376770abd3a36b20394c5664afef1194c801c3f05e42566f085ed24d002bb0", "0xb368d219438b7f3f"},
		{"0", "0x10900000000000000000000000000000000000000000000000000"},
		{"0x77676767676760000000000000001002e000000000000040000000e000000000", "0xfffc000000000000767676240000000000002b0576047"},
		{"0x767676767676000000000076000000000000005600000000000000000000", "0x767676767676000000000076000000760000"},
		{"0x8200000000000000000000000000000000000000000000000000000000000000", "0x8200000000000000fe000004000000ffff000000fffff700"},
		{"0xdac7fff9ffd9e1322626262626262600", "0xd021262626262626"},
		{"0x8000000000000001800000000000000080000000000000008000000000000000", "0x800000000000000080000000000000008000000000000000"},
		{"0x00e8e8e8e2000100000009ea02000000000000ff3ffffff80000001000220000", "0x00e8e8e8e2000100000009ea02000000000000ff3ffffff800000010002280ff"},
		{"0x000000c9700000000000000000023f00c00014ff000000000000000022300805", "0x00000000c9700000000000000000023f00c00014ff002c000000000000223108"},
		{"0x40000000fd000000db0000000000000000000000000000000000000000000001", "0x40000000fd000000db0000000000000000000040000000fd000000db000001"},
		{"0x40000000fd000000db0000000000000000000000000000000000000000000001", "0x40000000fd000000db0000000000000000000040000000fd000000db0000d3"},
		{"0x001f000000000000000000000000000000200000000100000000000000000000", "0x0000000000000000000100000000ffffffffffffffff0000000000002e000000"},
		{"0x7effffff80000000000000000000000000020000440000000000000000000001", "0x7effffff800000007effffff800000008000ff0000010000"},
		{"0x5fd8fffffffffffffffffffffffffffffc090000ce700004d0c9ffffff000001", "0x2ffffffffffffffffffffffffffffffffff000000030000"},
		{"0x62d8fffffffffffffffffffffffffffffc18000000000000000000ca00000001", "0x2ffffffffffffffffffffffffffffffffff200000000000"},
		{"0x7effffff8000000000000000000000000000000000000000d900000000000001", "0x7effffff8000000000000000000000000000000000008001"},
		{"0x0000000000000006400aff20ff00200004e7fd1eff08ffca0afd1eff08ffca0a", "0x00000000000000210000000000000022"},
		{"0x00000000000000000000000000000000000000000000006d5adef08547abf7eb", "0x000000000000000000013590cab83b779e708b533b0eef3561483ddeefc841f5"},
		{"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0x00e8e8e8e2000100000009ea02000000000000ff3ffffff80000001000220000", "0xffffffffffffffff7effffff800000007effffff800000008000ff0000010000"},
	}

	// A collection of interesting input values for ternary operators (addmod, mulmod).
	ternTestCases = [][3]string{
		{"0", "0", "0"},
		{"1", "0", "0"},
		{"1", "1", "0"},
		{"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe", "0"},
		{"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "3", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
		{"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "2"},
		{"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "1"},
		{"0xffffffffffffffffffffffffffffffff", "0xffffffffffffffffffffffffffffffff", "0xfffffffffffffffffffffffffffffffe00000000000000000000000000000002"},
		{"0xffffffffffffffffffffffffffffffff", "0xffffffffffffffffffffffffffffffff", "0xfffffffffffffffffffffffffffffffe00000000000000000000000000000001"},
	}
)

func hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

// toSatUint converts x to saturated uint value.
func toSatUint(x *Int) uint {
	maxUint := ^uint(0)
	z, overflow := x.Uint64WithOverflow()
	if overflow || z > uint64(maxUint) {
		return maxUint
	}
	return uint(z)
}

// bigToSatUint converts x to saturated uint value.
func bigToShiftAmount(x *big.Int) uint {
	max := uint(256) // 256 is enough to zero the result.
	if x.Cmp(new(big.Int).SetUint64(uint64(max))) > 0 {
		return max
	}
	return uint(x.Uint64())
}

func checkOverflow(b *big.Int, f *Int, overflow bool) error {
	max := big.NewInt(0).SetBytes(hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
	shouldOverflow := b.Cmp(max) > 0
	if overflow != shouldOverflow {
		return fmt.Errorf("Overflow should be %v, was %v\nf= %x\nb= %x\b", shouldOverflow, overflow, f, b)
	}
	return nil
}

func checkUnderflow(b *big.Int, f *Int, underflow bool) error {
	shouldUnderflow := b.Cmp(big.NewInt(0)) < 0
	if underflow != shouldUnderflow {
		return fmt.Errorf("Undeflow should be %v, was %v\nf= %x\nb= %x\b", shouldUnderflow, underflow, f, b)
	}
	return nil
}

func randNums() (*big.Int, *Int, error) {
	//How many bits? 0-256
	nbits, _ := rand.Int(rand.Reader, big.NewInt(256))
	//Max random value, a 130-bits integer, i.e 2^130
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(nbits.Int64()), nil)
	b, _ := rand.Int(rand.Reader, max)
	f, overflow := FromBig(b)
	err := checkOverflow(b, f, overflow)
	return b, f, err
}
func randHighNums() (*big.Int, *Int, error) {
	//How many bits? 0-256
	nbits := int64(256)
	//Max random value, a 130-bits integer, i.e 2^130
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(nbits), nil)
	//Generate cryptographically strong pseudo-random between 0 - max
	b, _ := rand.Int(rand.Reader, max)
	f, overflow := FromBig(b)
	//fmt.Printf("f %v\n", f.Hex())
	err := checkOverflow(b, f, overflow)
	return b, f, err
}
func checkEq(b *big.Int, f *Int) bool {
	f2, _ := FromBig(b)
	return f.Eq(f2)
}

func requireEq(t *testing.T, exp *big.Int, got *Int, txt string) bool {
	expF, _ := FromBig(exp)
	if !expF.Eq(got) {
		t.Errorf("got %x expected %x: %v\n", got, expF, txt)
		return false
	}
	return true
}

func testRandomOp(t *testing.T, nativeFunc func(a, b, c *Int), bigintFunc func(a, b, c *big.Int)) {
	for i := 0; i < 10000; i++ {
		b1, f1, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		b2, f2, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		f1a, f2a := f1.Clone(), f2.Clone()
		nativeFunc(f1, f1, f2)
		bigintFunc(b1, b1, b2)
		//checkOverflow(b, f1, overflow)
		if eq := checkEq(b1, f1); !eq {
			bf, _ := FromBig(b1)
			t.Fatalf("Expected equality:\nf1= %x\nf2= %x\n[ op ]==\nf = %x\nbf= %x\nb = %x\n", f1a, f2a, f1, bf, b1)
		}
	}
}
func TestRandomSubOverflow(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b, f1, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		b2, f2, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		f1a, f2a := f1.Clone(), f2.Clone()
		overflow := f1.SubOverflow(f1, f2)
		b.Sub(b, b2)
		if err := checkUnderflow(b, f1, overflow); err != nil {
			t.Fatal(err)
		}
		if eq := checkEq(b, f1); !eq {
			t.Fatalf("Expected equality:\nf1= %x\nf2= %x\n[ - ]==\nf= %x\nb= %x\n", f1a, f2a, f1, b)
		}
	}
}
func TestRandomSub(t *testing.T) {
	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.Sub(f2, f3)
		},
		func(b1, b2, b3 *big.Int) {
			b1.Sub(b2, b3)
		},
	)
}

func TestRandomAdd(t *testing.T) {
	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.Add(f2, f3)
		},
		func(b1, b2, b3 *big.Int) {
			b1.Add(b2, b3)
		},
	)
}
func TestRandomMul(t *testing.T) {

	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.Mul(f2, f3)
		},
		func(b1, b2, b3 *big.Int) {
			b1.Mul(b2, b3)
		},
	)
}
func TestRandomMulOverflow(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b, f1, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		b2, f2, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		f1a, f2a := f1.Clone(), f2.Clone()
		overflow := f1.MulOverflow(f1, f2)
		b.Mul(b, b2)
		if err := checkOverflow(b, f1, overflow); err != nil {
			t.Fatal(err)
		}
		if eq := checkEq(b, f1); !eq {
			t.Fatalf("Expected equality:\nf1= %x\nf2= %x\n[ - ]==\nf= %x\nb= %x\n", f1a, f2a, f1, b)
		}
	}
}
func TestRandomSquare(t *testing.T) {
	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.squared()
		},
		func(b1, b2, b3 *big.Int) {
			b1.Mul(b1, b1)
		},
	)
}
func TestRandomDiv(t *testing.T) {
	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.Div(f2, f3)
		},
		func(b1, b2, b3 *big.Int) {
			if b3.Sign() == 0 {
				b1.SetUint64(0)
			} else {
				b1.Div(b2, b3)
			}
		},
	)
}

func TestRandomMod(t *testing.T) {
	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.Mod(f2, f3)
		},
		func(b1, b2, b3 *big.Int) {
			if b3.Sign() == 0 {
				b1.SetUint64(0)
			} else {
				b1.Mod(b2, b3)
			}
		},
	)
}
func TestRandomSMod(t *testing.T) {
	testRandomOp(t,
		func(f1, f2, f3 *Int) {
			f1.SMod(f2, f3)
		},
		func(b1, b2, b3 *big.Int) {
			SMod(b1, b2, b3)
		},
	)
}

func TestRandomMulMod(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b1, f1, err := randNums()
		if err != nil {
			t.Fatalf("Error getting a random number: %v", err)
		}

		b2, f2, err := randNums()
		if err != nil {
			t.Fatalf("Error getting a random number: %v", err)
		}

		b3, f3, err := randNums()
		if err != nil {
			t.Fatalf("Error getting a random number: %v", err)
		}

		b4, f4, _ := randNums()
		for b4.Cmp(big.NewInt(0)) == 0 {
			b4, f4, err = randNums()
			if err != nil {
				t.Fatalf("Error getting a random number: %v", err)
			}
		}

		f1.MulMod(f2, f3, f4)
		b1.Mod(big.NewInt(0).Mul(b2, b3), b4)

		if !checkEq(b1, f1) {
			t.Fatalf("Expected equality:\nf2= %x\nf3= %x\nf4= %x\n[ op ]==\nf = %x\nb = %x\n", f2, f3, f4, f1, b1)
		}
	}
}

func S256(x *big.Int) *big.Int {
	if x.Cmp(bigtt255) < 0 {
		return x
	} else {
		return new(big.Int).Sub(x, bigtt256)
	}
}

func TestRandomAbs(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b, f1, err := randHighNums()
		if err != nil {
			t.Fatal(err)
		}
		U256(b)
		b2 := S256(big.NewInt(0).Set(b))
		b2.Abs(b2)
		f1a := new(Int).Abs(f1)

		if eq := checkEq(b2, f1a); !eq {
			bf, _ := FromBig(b2)
			t.Fatalf("Expected equality:\nf1= %x\n[ abs ]==\nf = %x\nbf= %x\nb = %x\n", f1, f1a, bf, b2)
		}
	}
}

func TestRandomSDiv(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b, f1, err := randHighNums()
		if err != nil {
			t.Fatal(err)
		}
		b2, f2, err := randHighNums()
		if err != nil {
			t.Fatal(err)
		}
		U256(b)
		U256(b2)

		f1a, f2a := f1.Clone(), f2.Clone()

		f1aAbs, f2aAbs := new(Int).Abs(f1), new(Int).Abs(f2)

		f1.SDiv(f1, f2)
		if b2.BitLen() == 0 {
			// zero
			b = big.NewInt(0)
		} else {
			b = SDiv(b, b, b2)
		}
		if eq := checkEq(b, f1); !eq {
			bf, _ := FromBig(b)
			t.Fatalf("Expected equality:\nf1  = %x\nf2  = %x\n\n\nabs1= %x\nabs2= %x\n[sdiv]==\nf   = %x\nbf  = %x\nb   = %x\n",
				f1a, f2a, f1aAbs, f2aAbs, f1, bf, b)
		}
	}
}

func TestRandomLsh(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b, f1, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		f1a := f1.Clone()
		nbits, _ := rand.Int(rand.Reader, big.NewInt(256))
		n := uint(nbits.Uint64())
		f1.Lsh(f1, n)
		b.Lsh(b, n)
		if eq := checkEq(b, f1); !eq {
			bf, _ := FromBig(b)
			t.Fatalf("Expected equality:\nf1= %x\n n= %v\n[ << ]==\nf = %x\nbf= %x\nb = %x\n", f1a, n, f1, bf, b)
		}
	}
}

func TestRandomRsh(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b, f1, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		f1a := f1.Clone()
		nbits, _ := rand.Int(rand.Reader, big.NewInt(256))
		n := uint(nbits.Uint64())
		f1.Rsh(f1, n)
		b.Rsh(b, n)
		if eq := checkEq(b, f1); !eq {
			t.Fatalf("Expected equality:\nf1= %x\n n= %v\n[ << ]==\nf= %x\nb= %x\n", f1a, n, f1, b)
		}
	}
}

func TestSRsh(t *testing.T) {
	type testCase struct {
		arg      string
		n        uint
		expected string
	}
	testCases := []testCase{
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 0, "FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 16, "FFFFFFFFEEEEDDDDCCCCBBBBAAAA999988887777666655554444333322221111"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 64, "FFFFFFFFFFFFFFFFFFFFEEEEDDDDCCCCBBBBAAAA999988887777666655554444"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 96, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFEEEEDDDDCCCCBBBBAAAA9999888877776666"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 127, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFDDDDBBBB99997777555533331110"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 128, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEEEEDDDDCCCCBBBBAAAA99998888"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 129, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF7776EEEE6665DDDD5554CCCC444"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 192, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEEEEDDDDCCCC"},
		{"8000000000000000000000000000000000000000000000000000000000000000", 254, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFE"},
		{"8000000000000000000000000000000000000000000000000000000000000000", 255, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 256, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
		{"FFFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 300, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
		{"7FFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 16, "7FFFEEEEDDDDCCCCBBBBAAAA999988887777666655554444333322221111"},
		{"7FFFEEEEDDDDCCCCBBBBAAAA9999888877776666555544443333222211110000", 256, ""},
	}

	for i := 0; i < len(testCases); i++ {
		tc := &testCases[i]
		arg := new(Int).SetBytes(hex2Bytes(tc.arg))
		argCopy := new(Int).Set(arg)
		expected := new(Int).SetBytes(hex2Bytes(tc.expected))
		result := new(Int).SRsh(arg, tc.n)

		if !result.Eq(expected) {
			t.Logf("args: %s, %d\n", tc.arg, tc.n)
			t.Logf("exp : %x\n", expected)
			t.Logf("got : %x\n\n", result)
			t.Fail()
		}

		if !arg.Eq(argCopy) {
			t.Errorf("Argument has been modified\n")
		}
	}
}

func TestByte(t *testing.T) {
	z := new(Int).SetBytes(hex2Bytes("ABCDEF09080706050403020100000000000000000000000000000000000000ef"))
	actual := z.Byte(NewInt().SetUint64(0))
	expected := new(Int).SetBytes(hex2Bytes("00000000000000000000000000000000000000000000000000000000000000ab"))
	if !actual.Eq(expected) {
		t.Fatalf("Expected %x, got %x", expected, actual)
	}

	z = new(Int).SetBytes(hex2Bytes("ABCDEF09080706050403020100000000000000000000000000000000000000ef"))
	actual = z.Byte(NewInt().SetUint64(31))
	expected = new(Int).SetBytes(hex2Bytes("00000000000000000000000000000000000000000000000000000000000000ef"))
	if !actual.Eq(expected) {
		t.Fatalf("Expected %x, got %x", expected, actual)
	}

	z = new(Int).SetBytes(hex2Bytes("ABCDEF09080706050403020100000000000000000000000000000000000000ef"))
	actual = z.Byte(NewInt().SetUint64(32))
	expected = new(Int).SetBytes(hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"))
	if !actual.Eq(expected) {
		t.Fatalf("Expected %x, got %x", expected, actual)
	}

	z = new(Int).SetBytes(hex2Bytes("ABCDEF0908070605040302011111111111111111111111111111111111111111"))
	actual = z.Byte(new(Int).SetBytes(hex2Bytes("f000000000000000000000000000000000000000000000000000000000000001")))
	expected = new(Int).SetBytes(hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"))
	if !actual.Eq(expected) {
		t.Fatalf("Expected %x, got %x", expected, actual)
	}
}

func TestSignExtend(t *testing.T) {
	type testCase struct {
		arg      string
		n        uint64
		expected string
	}
	testCases := []testCase{
		{"8080808080808080808080808080808080808080808080808080808080808080", 0, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80"},
		{"8080808080808080808080808080808080808080808080808080808080808080", 1, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8080"},
		{"8080808080808080808080808080808080808080808080808080808080808080", 2, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808080"},
		{"8080808080808080808080808080808080808080808080808080808080808080", 3, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffff80808080"},
		{"8080808080808080808080808080808080808080808080808080808080808080", 30, "ff80808080808080808080808080808080808080808080808080808080808080"},
		{"8080808080808080808080808080808080808080808080808080808080808080", 31, "8080808080808080808080808080808080808080808080808080808080808080"},
		{"8080808080808080808080808080808080808080808080808080808080808080", 32, "8080808080808080808080808080808080808080808080808080808080808080"},
		{"4040404040404040404040404040404040404040404040404040404040404040", 0, "40"},
		{"4040404040404040404040404040404040404040404040404040404040404040", 1, "4040"},
		{"4040404040404040404040404040404040404040404040404040404040404040", 30, "40404040404040404040404040404040404040404040404040404040404040"},
		{"4040404040404040404040404040404040404040404040404040404040404040", 31, "4040404040404040404040404040404040404040404040404040404040404040"},
		{"4040404040404040404040404040404040404040404040404040404040404040", 32, "4040404040404040404040404040404040404040404040404040404040404040"},
	}

	for i := 0; i < len(testCases); i++ {
		tc := &testCases[i]
		arg := new(Int).SetBytes(hex2Bytes(tc.arg))
		argCopy := new(Int).Set(arg)
		n := new(Int).SetUint64(tc.n)
		nCopy := new(Int).Set(n)
		expected := new(Int).SetBytes(hex2Bytes(tc.expected))
		result := new(Int).ExtendSign(arg, n)

		if !result.Eq(expected) {
			t.Logf("args: %s, %d\n", tc.arg, tc.n)
			t.Logf("exp : %x\n", expected)
			t.Logf("got : %x\n\n", result)
			t.Fail()
		}

		if !arg.Eq(argCopy) {
			t.Errorf("First argument has been modified\n")
		}

		if !n.Eq(nCopy) {
			t.Errorf("Second argument has been modified\n")
		}
	}
}

func TestSubUint64(t *testing.T) {
	type testCase struct {
		arg string
		n   uint64
	}
	testCases := []testCase{
		{"0", 1},
		{"1", 0},
		{"1", 1},
		{"1", 3},
		{"0x10000000000000000", 1},
		{"0x100000000000000000000000000000000", 1},
	}

	for i := 0; i < len(testCases); i++ {
		tc := &testCases[i]
		bigArg, _ := new(big.Int).SetString(tc.arg, 0)
		arg, _ := FromBig(bigArg)
		expected, _ := FromBig(U256(new(big.Int).Sub(bigArg, new(big.Int).SetUint64(tc.n))))
		result := new(Int).SubUint64(arg, tc.n)

		if !result.Eq(expected) {
			t.Logf("args: %s, %d\n", tc.arg, tc.n)
			t.Logf("exp : %x\n", expected)
			t.Logf("got : %x\n\n", result)
			t.Fail()
		}
	}
}

func TestSGT(t *testing.T) {

	x := new(Int).SetBytes(hex2Bytes("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"))
	y := new(Int).SetBytes(hex2Bytes("00"))
	actual := x.Sgt(y)
	if actual {
		t.Fatalf("Expected %v false", actual)
	}

	x = new(Int).SetBytes(hex2Bytes("00"))
	y = new(Int).SetBytes(hex2Bytes("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"))
	actual = x.Sgt(y)
	if !actual {
		t.Fatalf("Expected %v true", actual)
	}
}

const (
	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
)

var (
	tt256m1 = new(big.Int).Sub(bigtt256, big.NewInt(1))
)

// U256 encodes as a 256 bit two's complement number. This operation is destructive.
func U256(x *big.Int) *big.Int {
	return x.And(x, tt256m1)
}

// Exp implements exponentiation by squaring.
// The result is truncated to 256 bits.
func Exp(result, base, exponent *big.Int) *big.Int {
	result.SetUint64(1)

	for _, word := range exponent.Bits() {
		for i := 0; i < wordBits; i++ {
			if word&1 == 1 {
				U256(result.Mul(result, base))
			}
			U256(base.Mul(base, base))
			word >>= 1
		}
	}
	return result
}

func SDiv(result, x, y *big.Int) *big.Int {
	if y.Sign() == 0 {
		return result.SetUint64(0)
	}
	sx := S256(x)
	sy := S256(y)

	n := new(big.Int)
	if sx.Sign() == sy.Sign() {
		n.SetInt64(1)
	} else {
		n.SetInt64(-1)
	}
	result.Div(sx.Abs(sx), sy.Abs(sy))
	result.Mul(result, n)
	return result
}

func SMod(result, x, y *big.Int) *big.Int {
	if y.Sign() == 0 {
		return result.SetUint64(0)
	}

	sx := S256(x)
	sy := S256(y)
	neg := sx.Sign() < 0

	result.Mod(sx.Abs(sx), sy.Abs(sy))
	if neg {
		result.Neg(result)
	}
	return U256(result)
}

func addMod(result, x, y, mod *big.Int) *big.Int {
	return result.Mod(result.Add(x, y), mod)
}

func mulMod(result, x, y, mod *big.Int) *big.Int {
	return result.Mod(result.Mul(x, y), mod)
}

func referenceExp(base, exponent *big.Int) *big.Int {
	// TODO: Maybe use the Exp() procedure from above?
	res := new(big.Int)
	return res.Exp(base, exponent, bigtt256)
}

func TestRandomExp(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b_base, base, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		b_exp, exp, err := randNums()
		if err != nil {
			t.Fatal(err)
		}
		basecopy, expcopy := base.Clone(), exp.Clone()

		f_res, overflow := FromBig(referenceExp(base.ToBig(), exp.ToBig()))
		if overflow {
			t.Fatal("FromBig(exp) overflow")
		}

		b_res := Exp(new(big.Int), b_base, b_exp)
		if eq := checkEq(b_res, f_res); !eq {
			bf, _ := FromBig(b_res)
			t.Fatalf("Expected equality:\nbase= %x\nexp = %x\n[ ^ ]==\nf = %x\nbf= %x\nb = %x\n", basecopy, expcopy, f_res, bf, b_res)
		}
	}
}

func TestUnOp(t *testing.T) {
	proc := func(t *testing.T, op func(a, b *Int) *Int, bigOp func(a, b *big.Int) *big.Int) {
		for i := 0; i < len(unTestCases); i++ {
			b1, _ := new(big.Int).SetString(unTestCases[i], 0)
			f1orig, _ := FromBig(b1)
			f1 := new(Int).Set(f1orig)

			// Compare result with big.Int.
			expected, _ := FromBig(bigOp(new(big.Int), b1))
			result := op(new(Int), f1)
			if !result.Eq(expected) {
				t.Logf("arg : %s\n", unTestCases[i])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}

			// Check if arguments are unmodified.
			if !f1.Eq(f1orig) {
				t.Logf("arg : %s\n", unTestCases[i])
				t.Errorf("first argument had been modified: %x\n", f1)
			}

			// Check if reusing args as result works correctly.
			result = op(f1, f1)
			if result != f1 {
				t.Logf("arg : %s\n", unTestCases[i])
				t.Errorf("unexpected pointer returned: %p, expected: %p\n", result, f1)
			}
			if !result.Eq(expected) {
				t.Logf("arg : %s\n", unTestCases[i])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}
		}
	}

	t.Run("Not", func(t *testing.T) { proc(t, (*Int).Not, (*big.Int).Not) })
	t.Run("Neg", func(t *testing.T) { proc(t, (*Int).Neg, (*big.Int).Neg) })
}

func TestBinOp(t *testing.T) {
	proc := func(t *testing.T, op func(a, b, c *Int) *Int, bigOp func(a, b, c *big.Int) *big.Int) {
		for i := 0; i < len(binTestCases); i++ {
			b1, _ := new(big.Int).SetString(binTestCases[i][0], 0)
			b2, _ := new(big.Int).SetString(binTestCases[i][1], 0)
			f1orig, _ := FromBig(b1)
			f2orig, _ := FromBig(b2)
			f1 := new(Int).Set(f1orig)
			f2 := new(Int).Set(f2orig)

			// Compare result with big.Int.
			expected, _ := FromBig(bigOp(new(big.Int), b1, b2))
			result := op(new(Int), f1, f2)
			if !result.Eq(expected) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}

			// Check if arguments are unmodified.
			if !f1.Eq(f1orig) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Errorf("first argument had been modified: %x\n", f1)
			}
			if !f2.Eq(f2orig) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Errorf("second argument had been modified: %x\n", f2)
			}

			// Check if reusing args as result works correctly.
			result = op(f1, f1, f2orig)
			if result != f1 {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Errorf("unexpected pointer returned: %p, expected: %p\n", result, f1)
			}
			if !result.Eq(expected) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}
			result = op(f2, f1orig, f2)
			if result != f2 {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Errorf("unexpected pointer returned: %p, expected: %p\n", result, f2)
			}
			if !result.Eq(expected) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}
		}
	}

	t.Run("Add", func(t *testing.T) { proc(t, (*Int).Add, (*big.Int).Add) })
	t.Run("Sub", func(t *testing.T) { proc(t, (*Int).Sub, (*big.Int).Sub) })
	t.Run("Mul", func(t *testing.T) { proc(t, (*Int).Mul, (*big.Int).Mul) })
	t.Run("Div", func(t *testing.T) {
		proc(t, (*Int).Div, func(z, x, y *big.Int) *big.Int {
			if y.Sign() == 0 {
				return z.SetUint64(0)
			}
			return z.Div(x, y)
		})
	})
	t.Run("Mod", func(t *testing.T) {
		proc(t, (*Int).Mod, func(z, x, y *big.Int) *big.Int {
			if y.Sign() == 0 {
				return z.SetUint64(0)
			}
			return z.Mod(x, y)
		})
	})
	t.Run("SDiv", func(t *testing.T) { proc(t, (*Int).SDiv, SDiv) })
	t.Run("SMod", func(t *testing.T) { proc(t, (*Int).SMod, SMod) })
	t.Run("Exp", func(t *testing.T) { proc(t, (*Int).Exp, Exp) })

	t.Run("And", func(t *testing.T) { proc(t, (*Int).And, (*big.Int).And) })
	t.Run("Or", func(t *testing.T) { proc(t, (*Int).Or, (*big.Int).Or) })
	t.Run("Xor", func(t *testing.T) { proc(t, (*Int).Xor, (*big.Int).Xor) })

	t.Run("Lsh", func(t *testing.T) {
		proc(t, func(z, x, y *Int) *Int {
			return z.Lsh(x, toSatUint(y))
		}, func(z, x, y *big.Int) *big.Int {
			return z.Lsh(x, bigToShiftAmount(y))
		})
	})
	t.Run("Rsh", func(t *testing.T) {
		proc(t, func(z, x, y *Int) *Int {
			return z.Rsh(x, toSatUint(y))
		}, func(z, x, y *big.Int) *big.Int {
			return z.Rsh(x, bigToShiftAmount(y))
		})
	})
}

func TestTernOp(t *testing.T) {
	proc := func(t *testing.T, op func(a, b, c, d *Int) *Int, bigOp func(a, b, c, d *big.Int) *big.Int) {
		for i := 0; i < len(ternTestCases); i++ {
			b1, _ := new(big.Int).SetString(ternTestCases[i][0], 0)
			b2, _ := new(big.Int).SetString(ternTestCases[i][1], 0)
			b3, _ := new(big.Int).SetString(ternTestCases[i][2], 0)
			f1orig, _ := FromBig(b1)
			f2orig, _ := FromBig(b2)
			f3orig, _ := FromBig(b3)
			f1 := new(Int).Set(f1orig)
			f2 := new(Int).Set(f2orig)
			f3 := new(Int).Set(f3orig)

			// Compare result with big.Int.
			expected, _ := FromBig(bigOp(new(big.Int), b1, b2, b3))
			result := op(new(Int), f1, f2, f3)
			if !result.Eq(expected) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}

			// Check if arguments are unmodified.
			if !f1.Eq(f1orig) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Errorf("first argument had been modified: %x\n", f1)
			}
			if !f2.Eq(f2orig) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Errorf("second argument had been modified: %x\n", f2)
			}
			if !f3.Eq(f3orig) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Errorf("third argument had been modified: %x\n", f3)
			}

			// Check if reusing args as result works correctly.
			result = op(f1, f1, f2orig, f3orig)
			if result != f1 {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Errorf("unexpected pointer returned: %p, expected: %p\n", result, f1)
			}
			if !result.Eq(expected) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}
			result = op(f2, f1orig, f2, f3orig)
			if result != f2 {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Errorf("unexpected pointer returned: %p, expected: %p\n", result, f2)
			}
			if !result.Eq(expected) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}
			result = op(f3, f1orig, f2orig, f3)
			if result != f3 {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Errorf("unexpected pointer returned: %p, expected: %p\n", result, f3)
			}
			if !result.Eq(expected) {
				t.Logf("args: %s, %s, %s\n", ternTestCases[i][0], ternTestCases[i][1], ternTestCases[i][2])
				t.Logf("exp : %x\n", expected)
				t.Logf("got : %x\n\n", result)
				t.Fail()
			}
		}
	}
	t.Run("AddMod", func(t *testing.T) {
		proc(t, (*Int).AddMod, func(z, x, y, m *big.Int) *big.Int {
			if m.Sign() == 0 {
				return z.SetUint64(0)
			}
			return addMod(z, x, y, m)
		})
	})
	t.Run("MulMod", func(t *testing.T) {
		proc(t, (*Int).MulMod, func(z, x, y, m *big.Int) *big.Int {
			if m.Sign() == 0 {
				return z.SetUint64(0)
			}
			return mulMod(z, x, y, m)
		})
	})
}

func TestCmpOp(t *testing.T) {
	proc := func(t *testing.T, op func(a, b *Int) bool, bigOp func(a, b *big.Int) bool) {
		for i := 0; i < len(binTestCases); i++ {
			b1, _ := new(big.Int).SetString(binTestCases[i][0], 0)
			b2, _ := new(big.Int).SetString(binTestCases[i][1], 0)
			f1orig, _ := FromBig(b1)
			f2orig, _ := FromBig(b2)
			f1 := new(Int).Set(f1orig)
			f2 := new(Int).Set(f2orig)

			// Compare result with big.Int.
			expected := bigOp(b1, b2)
			result := op(f1, f2)
			if result != expected {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Logf("exp : %t\n", expected)
				t.Logf("got : %t\n\n", result)
				t.Fail()
			}

			// Check if arguments are unmodified.
			if !f1.Eq(f1orig) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Errorf("first argument had been modified: %x\n", f1)
			}
			if !f2.Eq(f2orig) {
				t.Logf("args: %s, %s\n", binTestCases[i][0], binTestCases[i][1])
				t.Errorf("second argument had been modified: %x\n", f2)
			}
		}
	}

	t.Run("Eq", func(t *testing.T) { proc(t, (*Int).Eq, func(a, b *big.Int) bool { return a.Cmp(b) == 0 }) })
	t.Run("Lt", func(t *testing.T) { proc(t, (*Int).Lt, func(a, b *big.Int) bool { return a.Cmp(b) < 0 }) })
	t.Run("Gt", func(t *testing.T) { proc(t, (*Int).Gt, func(a, b *big.Int) bool { return a.Cmp(b) > 0 }) })
	t.Run("SLt", func(t *testing.T) { proc(t, (*Int).Slt, func(a, b *big.Int) bool { return S256(a).Cmp(S256(b)) < 0 }) })
	t.Run("SGt", func(t *testing.T) { proc(t, (*Int).Sgt, func(a, b *big.Int) bool { return S256(a).Cmp(S256(b)) > 0 }) })
	t.Run("CmpEq", func(t *testing.T) {
		proc(t, func(a, b *Int) bool { return a.Cmp(b) == 0 }, func(a, b *big.Int) bool { return a.Cmp(b) == 0 })
	})
	t.Run("CmpLt", func(t *testing.T) {
		proc(t, func(a, b *Int) bool { return a.Cmp(b) < 0 }, func(a, b *big.Int) bool { return a.Cmp(b) < 0 })
	})
	t.Run("CmpGt", func(t *testing.T) {
		proc(t, func(a, b *Int) bool { return a.Cmp(b) > 0 }, func(a, b *big.Int) bool { return a.Cmp(b) > 0 })
	})

	t.Run("LtUint64", func(t *testing.T) {
		proc(t, func(a, b *Int) bool {
			return a.LtUint64(b.Uint64())
		}, func(a, b *big.Int) bool {
			return a.Cmp(new(big.Int).SetUint64(b.Uint64())) < 0
		})
	})
	t.Run("GtUint64", func(t *testing.T) {
		proc(t, func(a, b *Int) bool {
			return a.GtUint64(b.Uint64())
		}, func(a, b *big.Int) bool {
			return a.Cmp(new(big.Int).SetUint64(b.Uint64())) > 0
		})
	})
}

// TestFixedExpReusedArgs tests the cases in Exp() where the arguments (including result) alias the same objects.
func TestFixedExpReusedArgs(t *testing.T) {
	f2 := Int{2, 0, 0, 0}
	f2.Exp(&f2, &f2)
	requireEq(t, big.NewInt(2*2), &f2, "")

	// TODO: This is tested in TestBinOp().
	f3 := Int{3, 0, 0, 0}
	f4 := Int{4, 0, 0, 0}
	f3.Exp(&f4, &f3)
	requireEq(t, big.NewInt(4*4*4), &f3, "")

	// TODO: This is tested in TestBinOp().
	f5 := Int{5, 0, 0, 0}
	f6 := Int{6, 0, 0, 0}
	f6.Exp(&f6, &f5)
	requireEq(t, big.NewInt(6*6*6*6*6), &f6, "")

	f3 = Int{3, 0, 0, 0}
	fr := new(Int).Exp(&f3, &f3)
	requireEq(t, big.NewInt(3*3*3), fr, "")
}

func TestByteRepresentation(t *testing.T) {
	a := big.NewInt(0xFF0AFcafe)
	aa := new(Int).SetUint64(0xFF0afcafe)
	bb := new(Int).SetBytes(a.Bytes())
	if !aa.Eq(bb) {
		t.Fatal("aa != bb")
	}

	check := func(padded []byte, expectedHex string) {
		if expected := hex2Bytes(expectedHex); !bytes.Equal(padded, expected) {
			t.Errorf("incorrect padded bytes: %x, expected: %x", padded, expected)
		}
	}

	check(aa.PaddedBytes(32), "0000000000000000000000000000000000000000000000000000000ff0afcafe")
	check(aa.PaddedBytes(20), "0000000000000000000000000000000ff0afcafe")
	check(aa.PaddedBytes(40), "00000000000000000000000000000000000000000000000000000000000000000000000ff0afcafe")

	bytearr := hex2Bytes("0e320219838e859b2f9f18b72e3d4073ca50b37d")
	a = new(big.Int).SetBytes(bytearr)
	aa = new(Int).SetBytes(bytearr)
	bb = new(Int).SetBytes(a.Bytes())
	if !aa.Eq(bb) {
		t.Fatal("aa != bb")
	}

	check(aa.PaddedBytes(32), "0000000000000000000000000e320219838e859b2f9f18b72e3d4073ca50b37d")
	check(aa.PaddedBytes(20), "0e320219838e859b2f9f18b72e3d4073ca50b37d")
	check(aa.PaddedBytes(40), "00000000000000000000000000000000000000000e320219838e859b2f9f18b72e3d4073ca50b37d")
}

func TestWriteToSlice(t *testing.T) {
	x1 := hex2Bytes("fe7fb0d1f59dfe9492ffbf73683fd1e870eec79504c60144cc7f5fc2bad1e611")

	a := big.NewInt(0).SetBytes(x1)
	fa, _ := FromBig(a)

	dest := make([]byte, 32)
	fa.WriteToSlice(dest)
	if !bytes.Equal(dest, x1) {
		t.Errorf("got %x, expected %x", dest, x1)
	}

	fb := NewInt()
	exp := make([]byte, 32)
	fb.WriteToSlice(dest)
	if !bytes.Equal(dest, exp) {
		t.Errorf("got %x, expected %x", dest, exp)
	}
	// a too small buffer
	// Should fill the lower parts, masking upper bytes
	exp = hex2Bytes("683fd1e870eec79504c60144cc7f5fc2bad1e611")
	dest = make([]byte, 20)
	fa.WriteToSlice(dest)
	if !bytes.Equal(dest, exp) {
		t.Errorf("got %x, expected %x", dest, exp)
	}

	// a too large buffer, already filled with stuff
	// Should fill the leftmost 32 bytes, not touch the other things
	dest = hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	exp = hex2Bytes("fe7fb0d1f59dfe9492ffbf73683fd1e870eec79504c60144cc7f5fc2bad1e611ffffffffffffffff")

	fa.WriteToSlice(dest)
	if !bytes.Equal(dest, exp) {
		t.Errorf("got %x, expected %x", dest, x1)
	}

	// an empty slice, no panics please
	dest = []byte{}
	exp = []byte{}

	fa.WriteToSlice(dest)
	if !bytes.Equal(dest, exp) {
		t.Errorf("got %x, expected %x", dest, x1)
	}

}
func TestInt_WriteToArray(t *testing.T) {
	x1 := hex2Bytes("0000000000000000000000000000d1e870eec79504c60144cc7f5fc2bad1e611")
	a := big.NewInt(0).SetBytes(x1)
	fa, _ := FromBig(a)

	{
		dest := [20]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		fa.WriteToArray20(&dest)
		exp := hex2Bytes("0000d1e870eec79504c60144cc7f5fc2bad1e611")
		if !bytes.Equal(dest[:], exp) {
			t.Errorf("got %x, expected %x", dest, exp)
		}

	}

	{
		dest := [32]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		fa.WriteToArray32(&dest)
		exp := hex2Bytes("0000000000000000000000000000d1e870eec79504c60144cc7f5fc2bad1e611")
		if !bytes.Equal(dest[:], exp) {
			t.Errorf("got %x, expected %x", dest, exp)
		}

	}
}

type gethAddress [20]byte

// SetBytes sets the address to the value of b.
// If b is larger than len(a) it will panic.
func (a *gethAddress) setBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-20:]
	}
	copy(a[20-len(b):], b)
}

// BytesToAddress returns Address with value b.
// If b is larger than len(h), b will be cropped from the left.
func bytesToAddress(b []byte) gethAddress {
	var a gethAddress
	a.setBytes(b)
	return a
}

type gethHash [32]byte

// SetBytes sets the address to the value of b.
// If b is larger than len(a) it will panic.
func (a *gethHash) setBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-32:]
	}
	copy(a[32-len(b):], b)
}

// BytesToHash returns gethHash with value b.
// If b is larger than len(h), b will be cropped from the left.
func bytesToHash(b []byte) gethHash {
	var a gethHash
	a.setBytes(b)
	return a
}

func TestByte20Representation(t *testing.T) {
	for i, tt := range []string{
		"1337fafafa0e320219838e859b2f9f18b72e3d4073ca50b37d",
		"fafafa0e320219838e859b2f9f18b72e3d4073ca50b37d",
		"0e320219838e859b2f9f18b72e3d4073ca50b37d",
		"320219838e859b2f9f18b72e3d4073ca50b37d",
		"838e859b2f9f18b72e3d4073ca50b37d",
		"38e859b2f9f18b72e3d4073ca50b37d",
		"f18b72e3d4073ca50b37d",
		"b37d",
		"01",
		"",
		"00",
	} {
		bytearr := hex2Bytes(tt)
		// big.Int -> address
		a := big.NewInt(0).SetBytes(bytearr)
		exp := bytesToAddress(a.Bytes())

		// uint256.Int -> address
		b := NewInt().SetBytes(bytearr)
		got := gethAddress(b.Bytes20())

		if got != exp {
			t.Errorf("testcase %d: got %x exp %x", i, got, exp)
		}
	}
}

func TestByte32Representation(t *testing.T) {
	for i, tt := range []string{
		"1337fafafa0e320219838e859b2f9f18b72e3d4073ca50b37d",
		"fafafa0e320219838e859b2f9f18b72e3d4073ca50b37d",
		"0e320219838e859b2f9f18b72e3d4073ca50b37d",
		"320219838e859b2f9f18b72e3d4073ca50b37d",
		"838e859b2f9f18b72e3d4073ca50b37d",
		"38e859b2f9f18b72e3d4073ca50b37d",
		"f18b72e3d4073ca50b37d",
		"b37d",
		"01",
		"",
		"00",
	} {
		bytearr := hex2Bytes(tt)
		// big.Int -> hash
		a := big.NewInt(0).SetBytes(bytearr)
		exp := bytesToHash(a.Bytes())

		// uint256.Int -> address
		b := NewInt().SetBytes(bytearr)
		got := gethHash(b.Bytes32())

		if got != exp {
			t.Errorf("testcase %d: got %x exp %x", i, got, exp)
		}
	}
}
