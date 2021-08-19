package bcrp

import (
	"encoding/hex"
	"testing"
)

func TestIsBCRPScript(t *testing.T) {
	tests := []struct {
		program  string
		expected bool
	}{
		{
			program:  "",
			expected: false,
		},
		{
			program:  "ae20ac20f5cdb9ada2ae9836bcfff32126d6b885aa3f73ee111a95d1bf37f3904aca5151ad",
			expected: false,
		},
		{
			// not OP_FAIL
			program:  "69046263727001012820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0",
			expected: false,
		},
		{
			// not bcrp
			program:  "6a044243525001012820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0",
			expected: false,
		},
		{
			// not version 1
			program:  "6a046263727001022820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0",
			expected: false,
		},
		{
			// p2wpkh script
			program:  "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			expected: false,
		},
		{
			// p2wsh script
			program:  "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			expected: false,
		},
		{
			// len(contract) == 0
			program:  "6a046263727001016a",
			expected: false,
		},
		{
			// len(contract) == 1
			program:  "6a04626372700101016a",
			expected: true,
		},
		{
			// 1 < len(contract) < 75
			program:  "6a046263727001012820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0",
			expected: true,
		},
		{
			// len(contract) == 75
			program:  "6a046263727001014b20e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403",
			expected: true,
		},
		{
			// 75 < len(contract) < 256
			program:  "6a046263727001014c9620e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403",
			expected: true,
		},
		{
			// len(contract) == 256
			program:  "6a046263727001014d000120e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e",
			expected: true,
		},
		{
			// len(contract) > 256
			program:  "6a046263727001014d2c0120e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403",
			expected: true,
		},
	}

	for i, test := range tests {
		program, err := hex.DecodeString(test.program)
		if err != nil {
			t.Fatal(err)
		}

		expected := IsBCRPScript(program)
		if expected != test.expected {
			t.Errorf("TestIsTemplateRegister #%d failed: got %v want %v", i, expected, test.expected)
		}
	}
}

func TestIsCallContractScript(t *testing.T) {
	tests := []struct {
		program  string
		expected bool
	}{
		{
			program:  "",
			expected: false,
		},
		{
			program:  "6a4c04626372704c01014c2820e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c0",
			expected: false,
		},
		{
			program:  "00204e4f02d43bf50171f7f25d046b7f016002da410fc00d2e8902e7b170c98cf946",
			expected: false,
		},
		{
			program:  "51204e4f02d43bf50171f7f25d046b7f016002da410fc00d2e8902e7b170c98cf946",
			expected: false,
		},
		{
			// p2wpkh script
			program:  "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			expected: false,
		},
		{
			// p2wsh script
			program:  "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			expected: false,
		},
		{
			program:  "0462637270204e4f02d43bf50171f7f25d046b7f016002da410fc00d2e8902e7b170c98cf946",
			expected: true,
		},
	}

	for i, test := range tests {
		program, err := hex.DecodeString(test.program)
		if err != nil {
			t.Fatal(err)
		}

		expected := IsCallContractScript(program)
		if expected != test.expected {
			t.Errorf("TestIsCallContractScript #%d failed: got %v want %v", i, expected, test.expected)
		}
	}
}

func TestParseContract(t *testing.T) {
	tests := []struct {
		program  string
		expected string
	}{
		{
			// BCRP script format: OP_FAIL + OP_DATA_4 + "bcrp" + OP_DATA_1 + "1" + {{dynamic_op}} + contract
			program:  "6a04626372700101100164740a52797b937b788791698700c0",
			expected: "0164740a52797b937b788791698700c0",
		},
		{
			// BCRP script format: OP_FAIL + OP_DATA_4 + "bcrp" + OP_DATA_1 + "1" + {{dynamic_op}} + contract
			program:  "6a046263727001014d2c0120e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403",
			expected: "20e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e78740320e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403ae7cac00c020e9108d3ca8049800727f6a3505b3a2710dc579405dde03c250f16d9a7e1e6e787403",
		},
	}

	for i, test := range tests {
		program, err := hex.DecodeString(test.program)
		if err != nil {
			t.Fatal(err)
		}

		contract, err := ParseContract(program)
		if err != nil {
			t.Fatal(err)
		}

		expected := hex.EncodeToString(contract[:])
		if expected != test.expected {
			t.Errorf("TestParseContract #%d failed: got %v want %v", i, expected, test.expected)
		}
	}
}

func TestParseContractHash(t *testing.T) {
	tests := []struct {
		program  string
		expected string
	}{
		{
			// call contract script format: OP_DATA_4 + "bcrp"+ OP_DATA_32 + SHA3-256(contract)
			program:  "0462637270204e4f02d43bf50171f7f25d046b7f016002da410fc00d2e8902e7b170c98cf946",
			expected: "4e4f02d43bf50171f7f25d046b7f016002da410fc00d2e8902e7b170c98cf946",
		},
	}

	for i, test := range tests {
		program, err := hex.DecodeString(test.program)
		if err != nil {
			t.Fatal(err)
		}

		hash, err := ParseContractHash(program)
		if err != nil {
			t.Fatal(err)
		}

		expected := hex.EncodeToString(hash[:])
		if expected != test.expected {
			t.Errorf("TestParseContractHash #%d failed: got %v want %v", i, expected, test.expected)
		}
	}
}
