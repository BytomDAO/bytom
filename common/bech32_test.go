// Copyright (c) 2017 Takatoshi Nakagawa
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
package common

import (
	"reflect"
	"strings"
	"testing"
)

func ScriptPubkey(version int, program []int) []int {
	if version != 0 {
		version += 0x50
	}
	return append(append([]int{version}, len(program)), program...)
}

var validChecksum = []string{
	"A12UEL5L",
	"an83characterlonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio1tt5tgs",
	"abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw",
	"11qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqc8247j",
	"split1checkupstagehandshakeupstreamerranterredcaperred2y9e3w",
}

type item struct {
	address      string
	scriptpubkey []int
}

var validAddress = []item{
	item{"BC1QW508D6QEJXTDG4Y5R3ZARVARY0C5XW7KV8F3T4",
		[]int{
			0x00, 0x14, 0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54,
			0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6,
		},
	},
	item{"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7",
		[]int{
			0x00, 0x20, 0x18, 0x63, 0x14, 0x3c, 0x14, 0xc5, 0x16, 0x68, 0x04,
			0xbd, 0x19, 0x20, 0x33, 0x56, 0xda, 0x13, 0x6c, 0x98, 0x56, 0x78,
			0xcd, 0x4d, 0x27, 0xa1, 0xb8, 0xc6, 0x32, 0x96, 0x04, 0x90, 0x32,
			0x62,
		},
	},
	item{"bc1pw508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7k7grplx",
		[]int{
			0x51, 0x28, 0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54,
			0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6,
			0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54, 0x94, 0x1c,
			0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6,
		},
	},
	item{"BC1SW50QA3JX3S",
		[]int{
			0x60, 0x02, 0x75, 0x1e,
		},
	},
	item{"bc1zw508d6qejxtdg4y5r3zarvaryvg6kdaj",
		[]int{
			0x52, 0x10, 0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54,
			0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23,
		},
	},
	item{"tb1qqqqqp399et2xygdj5xreqhjjvcmzhxw4aywxecjdzew6hylgvsesrxh6hy",
		[]int{
			0x00, 0x20, 0x00, 0x00, 0x00, 0xc4, 0xa5, 0xca, 0xd4, 0x62, 0x21,
			0xb2, 0xa1, 0x87, 0x90, 0x5e, 0x52, 0x66, 0x36, 0x2b, 0x99, 0xd5,
			0xe9, 0x1c, 0x6c, 0xe2, 0x4d, 0x16, 0x5d, 0xab, 0x93, 0xe8, 0x64,
			0x33,
		},
	},
}

var invalidAddress = []string{
	"tc1qw508d6qejxtdg4y5r3zarvary0c5xw7kg3g4ty",
	"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t5",
	"BC13W508D6QEJXTDG4Y5R3ZARVARY0C5XW7KN40WF2",
	"bc1rw5uspcuh",
	"bc10w508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7kw5rljs90",
	"BC1QR508D6QEJXTDG4Y5R3ZARVARYV98GJ9P",
	"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sL5k7",
	"tb1pw508d6qejxtdg4y5r3zarqfsj6c3",
	"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3pjxtptv",
}

func TestValidChecksum(t *testing.T) {
	for _, test := range validChecksum {
		hrp, data, err := Decode(test)
		if err != nil {
			t.Errorf("Valid checksum for %s : FAIL / error %+v\n", test, err)
		} else {
			t.Logf("Valid checksum for %s : ok / hrp : %+v , data : %+v\n", test, hrp, data)
		}
	}
}

func TestValidAddress(t *testing.T) {
	for _, test := range validAddress {
		hrp := "bc"
		version, program, err := AddressDecode(hrp, test.address)
		if err != nil {
			hrp = "tb"
			version, program, err = AddressDecode(hrp, test.address)
		}
		ok := err == nil
		if ok {
			output := ScriptPubkey(version, program)
			ok = reflect.DeepEqual(output, test.scriptpubkey)
		}
		if ok {
			recreate, err := AddressEncode(hrp, version, program)
			if err == nil {
				ok = recreate == strings.ToLower(test.address)
			}
		}
		if ok {
			t.Logf("Valid address %v : ok\n", test.address)
		} else {
			t.Errorf("Valid address %v : FAIL\n", test.address)
		}
	}
}

func TestInvalidAddress(t *testing.T) {
	for _, test := range invalidAddress {
		_, _, bcErr := AddressDecode("bc", test)
		t.Logf("bc error:%v\n", bcErr)
		_, _, tbErr := AddressDecode("tb", test)
		t.Logf("tb error:%v\n", tbErr)
		if bcErr != nil && tbErr != nil {
			t.Logf("Invalid address %v : ok\n", test)
		} else {
			t.Errorf("Invalid address %v : FAIL\n", test)
		}
	}
}

// add coverage tests

func TestCoverage(t *testing.T) {
	var err error
	var bech32String string
	var hrp string
	var data []int

	// AddressEncode
	bech32String, err = AddressEncode("bc", 1, []int{0, 1})
	if err != nil {
		t.Errorf("Coverage AddressEncode normal case : FAIL / error : %+v\n", err)
	} else {
		t.Log("Coverage AddressEncode normal case : ok / bech32String :", bech32String)
	}
	data = make([]int, 40)
	bech32String, err = AddressEncode("bc", 16, data)
	if err != nil {
		t.Errorf("Coverage AddressEncode normal case : FAIL / error : %+v\n", err)
	} else {
		t.Log("Coverage AddressEncode normal case : ok / bech32String :", bech32String)
	}
	data = make([]int, 20)
	bech32String, err = AddressEncode("bc", 0, data)
	if err != nil {
		t.Errorf("Coverage AddressEncode normal case : FAIL / error : %+v\n", err)
	} else {
		t.Log("Coverage AddressEncode normal case : ok / bech32String :", bech32String)
	}
	data = make([]int, 32)
	bech32String, err = AddressEncode("bc", 0, data)
	if err != nil {
		t.Errorf("Coverage AddressEncode normal case : FAIL / error : %+v\n", err)
	} else {
		t.Log("Coverage AddressEncode normal case : ok / bech32String :", bech32String)
	}
	data = make([]int, 1)
	_, err = AddressEncode("bc", 1, data)
	if err == nil {
		t.Errorf("Coverage AddressEncode invalid program length error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode invalid program length error case : ok / error :", err)
	}
	data = make([]int, 41)
	_, err = AddressEncode("bc", 1, data)
	if err == nil {
		t.Errorf("Coverage AddressEncode invalid program length error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode invalid program length error case : ok / error :", err)
	}
	data = make([]int, 26)
	_, err = AddressEncode("bc", 0, data)
	if err == nil {
		t.Errorf("Coverage AddressEncode invalid program length for witness version 0 (per BIP141) error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode invalid program length for witness version 0 (per BIP141) error case : ok / error :", err)
	}
	data = make([]int, 20)
	_, err = AddressEncode("Bc", 0, data)
	if err == nil {
		t.Errorf("Coverage AddressEncode Encode error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode Encode error case : ok / error :", err)
	}
	_, err = AddressEncode("bc", 1, []int{-1, 0})
	if err == nil {
		t.Errorf("Coverage AddressEncode invalid data range error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode invalid data range error case : ok / error :", err)
	}
	_, err = AddressEncode("bc", -1, data)
	if err == nil {
		t.Errorf("Coverage AddressEncode invalid witness version error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode invalid witness version error case : ok / error :", err)
	}
	_, err = AddressEncode("bc", 17, data)
	if err == nil {
		t.Errorf("Coverage AddressEncode invalid witness version error case : FAIL")
	} else {
		t.Log("Coverage AddressEncode invalid witness version error case : ok / error :", err)
	}

	// SegwitAddrDecode
	_, _, err = AddressDecode("a", "A12UEL5L")
	if err == nil {
		t.Errorf("Coverage SegwitAddrDecode invalid decode data length error case : FAIL")
	} else {
		t.Log("Coverage SegwitAddrDecode invalid decode data length error case : ok / error :", err)
	}

	// Decode
	_, _, err = Decode("!~1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqc356v3")
	if err != nil {
		t.Errorf("Coverage Decode normal case : FAIL / error :%v", err)
	} else {
		t.Log("Coverage Decode normal case : ok")
	}
	_, _, err = Decode("a1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq")
	if err == nil {
		t.Errorf("Coverage Decode too long error case : FAIL")
	} else {
		t.Log("Coverage Decode too long error case : ok / error :", err)
	}
	_, _, err = Decode("1")
	if err == nil {
		t.Errorf("Coverage Decode separator '1' at invalid position error case : FAIL")
	} else {
		t.Log("Coverage Decode separator '1' at invalid position error case : ok / error :", err)
	}
	_, _, err = Decode("a1qqqqq")
	if err == nil {
		t.Errorf("Coverage Decode separator '1' at invalid position error case : FAIL")
	} else {
		t.Log("Coverage Decode separator '1' at invalid position error case : ok / error :", err)
	}
	_, _, err = Decode("a" + string(32) + "1qqqqqq")
	if err == nil {
		t.Errorf("Coverage Decode invalid character human-readable part error case : FAIL")
	} else {
		t.Log("Coverage Decode invalid character human-readable part error case : ok / error :", err)
	}
	_, _, err = Decode("a" + string(127) + "1qqqqqq")
	if err == nil {
		t.Errorf("Coverage Decode invalid character human-readable part error case : FAIL")
	} else {
		t.Log("Coverage Decode invalid character human-readable part error case : ok / error :", err)
	}
	_, _, err = Decode("a1qqqqqb")
	if err == nil {
		t.Errorf("Coverage Decode invalid character data part error case : FAIL")
	} else {
		t.Log("Coverage Decode invalid character data part erroer case : ok / error :", err)
	}

	// Encode
	hrp = "bc"
	data = []int{}
	bech32String, err = Encode(hrp, data)
	if err != nil || bech32String != strings.ToLower(bech32String) {
		t.Errorf("Coverage Encode lower case : FAIL / bech32String : %v , error : %v", bech32String, err)
	} else {
		t.Log("Coverage Encode lower case : ok / bech32String : ", bech32String)
	}
	hrp = "BC"
	bech32String, err = Encode(hrp, data)
	if err != nil || bech32String != strings.ToUpper(bech32String) {
		t.Errorf("Coverage Encode upper case : FAIL / bech32String : %v , error : %v", bech32String, err)
	} else {
		t.Log("Coverage Encode upper case : ok / bech32String : ", bech32String)
	}
	hrp = "bc"
	data = make([]int, 90-7-len(hrp)+1)
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode too long error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode too long error case : ok / error : ", err)
	}
	hrp = ""
	data = make([]int, 90-7-len(hrp))
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode invalid hrp error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode invalid hrp error case : ok / error : ", err)
	}
	hrp = "Bc"
	data = make([]int, 90-7-len(hrp))
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode mix case error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode mix case error case : ok / error : ", err)
	}
	hrp = string(33) + string(126)
	data = make([]int, 90-7-len(hrp))
	bech32String, err = Encode(hrp, data)
	if err != nil {
		t.Errorf("Coverage Encode normal case : FAIL / error : %v", err)
	} else {
		t.Log("Coverage Encode normal case : ok / bech32String : ", bech32String)
	}
	hrp = string(32) + "c"
	data = make([]int, 90-7-len(hrp))
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode invalid character human-readable part error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode invalid character human-readable part error case : ok / error : ", err)
	}
	hrp = "b" + string(127)
	data = make([]int, 90-7-len(hrp))
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode invalid character human-readable part error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode invalid character human-readable part error case : ok / error : ", err)
	}
	hrp = "bc"
	data = []int{0, 31}
	bech32String, err = Encode(hrp, data)
	if err != nil {
		t.Errorf("Coverage Encode normal case : FAIL / error : %v", err)
	} else {
		t.Log("Coverage Encode normal case : ok / bech32String : ", bech32String)
	}
	hrp = "bc"
	data = []int{-1}
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode invalid data error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode invalid data error case : ok / error : ", err)
	}
	hrp = "bc"
	data = []int{32}
	bech32String, err = Encode(hrp, data)
	if err == nil {
		t.Errorf("Coverage Encode invalid data error case : FAIL / bech32String : %v", bech32String)
	} else {
		t.Log("Coverage Encode invalid data error case : ok / error : ", err)
	}
}
