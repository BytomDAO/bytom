package sm2

import (
	"encoding/hex"
	"testing"
)

func TestVerifyBytes(t *testing.T) {
	var tests = []struct {
		pubX []byte
		pubY []byte
		msg  []byte
		uid  []byte
		r    []byte
		s    []byte
	}{
		{
			pubX: mustDecodeHex("09f9df311e5421a150dd7d161e4bc5c672179fad1833fc076bb08ff356f35020"),
			pubY: mustDecodeHex("ccea490ce26775a52dc6ea718cc1aa600aed05fbf35e084a6632f6072da9ad13"),
			msg:  mustDecodeHex("6d65737361676520646967657374"),
			uid:  mustDecodeHex("31323334353637383132333435363738"),
			r:    mustDecodeHex("f5a03b0648d2c4630eeac513e1bb81a15944da3827d5b74143ac7eaceee720b3"),
			s:    mustDecodeHex("b1b6aa29df212fd8763182bc0d421ca1bb9038fd1f7f42d4840b69c485bbc1aa"),
		},
	}
	for i, tt := range tests {
		result := VerifyBytes(tt.pubX, tt.pubY, tt.msg, tt.uid, tt.r, tt.s)

		if !result {
			t.Errorf("result: %d is invalid!", i)
		}
	}
}

func BenchmarkVerifyBytes(b *testing.B) {
	var test = struct {
		pubX []byte
		pubY []byte
		msg  []byte
		uid  []byte
		r    []byte
		s    []byte
	}{
		pubX: mustDecodeHex("09f9df311e5421a150dd7d161e4bc5c672179fad1833fc076bb08ff356f35020"),
		pubY: mustDecodeHex("ccea490ce26775a52dc6ea718cc1aa600aed05fbf35e084a6632f6072da9ad13"),
		msg:  mustDecodeHex("6d65737361676520646967657374"),
		uid:  mustDecodeHex("31323334353637383132333435363738"),
		r:    mustDecodeHex("f5a03b0648d2c4630eeac513e1bb81a15944da3827d5b74143ac7eaceee720b3"),
		s:    mustDecodeHex("b1b6aa29df212fd8763182bc0d421ca1bb9038fd1f7f42d4840b69c485bbc1aa"),
	}
	for i := 0; i < b.N; i++ {
		VerifyBytes(test.pubX, test.pubY, test.msg, test.uid, test.r, test.s)
	}
}

func TestSm2VerifyBytes(t *testing.T) {
	var tests = []struct {
		publicKey []byte
		hash      []byte
		signature []byte
	}{
		{
			publicKey: mustDecodeHex("04" + "09f9df311e5421a150dd7d161e4bc5c672179fad1833fc076bb08ff356f35020" + "ccea490ce26775a52dc6ea718cc1aa600aed05fbf35e084a6632f6072da9ad13"),
			hash:      mustDecodeHex("f0b43e94ba45accaace692ed534382eb17e6ab5a19ce7b31f4486fdfc0d28640"),
			signature: mustDecodeHex("f5a03b0648d2c4630eeac513e1bb81a15944da3827d5b74143ac7eaceee720b3" + "b1b6aa29df212fd8763182bc0d421ca1bb9038fd1f7f42d4840b69c485bbc1aa"),
		},
	}
	for i, tt := range tests {
		result := Sm2VerifyBytes(tt.publicKey, tt.hash, tt.signature)

		if !result {
			t.Errorf("result: %d is invalid!", i)
		}
	}
}

func BenchmarkSm2VerifyBytes(b *testing.B) {
	var test = struct {
		publicKey []byte
		hash      []byte
		signature []byte
	}{

		publicKey: mustDecodeHex("04" + "09f9df311e5421a150dd7d161e4bc5c672179fad1833fc076bb08ff356f35020" + "ccea490ce26775a52dc6ea718cc1aa600aed05fbf35e084a6632f6072da9ad13"),
		hash:      mustDecodeHex("f0b43e94ba45accaace692ed534382eb17e6ab5a19ce7b31f4486fdfc0d28640"),
		signature: mustDecodeHex("f5a03b0648d2c4630eeac513e1bb81a15944da3827d5b74143ac7eaceee720b3" + "b1b6aa29df212fd8763182bc0d421ca1bb9038fd1f7f42d4840b69c485bbc1aa"),
	}
	for i := 0; i < b.N; i++ {
		Sm2VerifyBytes(test.publicKey, test.hash, test.signature)
	}
}

func TestVerifyCompressedPubkey(t *testing.T) {
	var tests = []struct {
		compressedPublicKey []byte
		hash                []byte
		signature           []byte
	}{
		{
			compressedPublicKey: mustDecodeHex("03" + "09f9df311e5421a150dd7d161e4bc5c672179fad1833fc076bb08ff356f35020"),
			hash:                mustDecodeHex("f0b43e94ba45accaace692ed534382eb17e6ab5a19ce7b31f4486fdfc0d28640"),
			signature:           mustDecodeHex("f5a03b0648d2c4630eeac513e1bb81a15944da3827d5b74143ac7eaceee720b3" + "b1b6aa29df212fd8763182bc0d421ca1bb9038fd1f7f42d4840b69c485bbc1aa"),
		},
	}
	for i, tt := range tests {
		result := VerifyCompressedPubkey(tt.compressedPublicKey, tt.hash, tt.signature)

		if !result {
			t.Errorf("result: %d is invalid!", i)
		}
	}
}

func BenchmarkVerifyCompressedPubkey(b *testing.B) {
	var test = struct {
		compressedPublicKey []byte
		hash                []byte
		signature           []byte
	}{
		compressedPublicKey: mustDecodeHex("03" + "09f9df311e5421a150dd7d161e4bc5c672179fad1833fc076bb08ff356f35020"),
		hash:                mustDecodeHex("f0b43e94ba45accaace692ed534382eb17e6ab5a19ce7b31f4486fdfc0d28640"),
		signature:           mustDecodeHex("f5a03b0648d2c4630eeac513e1bb81a15944da3827d5b74143ac7eaceee720b3" + "b1b6aa29df212fd8763182bc0d421ca1bb9038fd1f7f42d4840b69c485bbc1aa"),
	}
	for i := 0; i < b.N; i++ {
		VerifyCompressedPubkey(test.compressedPublicKey, test.hash, test.signature)
	}
}

// mustDecodeHex decode string to []byte
func mustDecodeHex(str string) []byte {
	data, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return data
}
