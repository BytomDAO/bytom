package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/testutil"
)

func TestReadWriteBlockWitness(t *testing.T) {
	cases := []struct {
		name      string
		witness   BlockWitness
		hexString string
	}{
		{
			name:      "normal block witness",
			witness:   testutil.MustDecodeHexString("e0776a3cf17b3e0f8340caeee32a75d02ecc25cf20bee9e5c7503bca3b2703c3c61fdcb4211ed59b58eb025ac81e06b138d54b5d01ea4614dd0f65e641836900"),
			hexString: "40e0776a3cf17b3e0f8340caeee32a75d02ecc25cf20bee9e5c7503bca3b2703c3c61fdcb4211ed59b58eb025ac81e06b138d54b5d01ea4614dd0f65e641836900",
		},
		{
			name:      "empty block witness",
			witness:   BlockWitness{},
			hexString: "00",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buff := []byte{}
			buffer := bytes.NewBuffer(buff)
			if err := c.witness.writeTo(buffer); err != nil {
				t.Fatal(err)
			}

			hexString := hex.EncodeToString(buffer.Bytes())
			if hexString != c.hexString {
				t.Errorf("test write block commitment fail, got:%s, want:%s", hexString, c.hexString)
			}

			blockWitness := &BlockWitness{}
			if err := blockWitness.readFrom(blockchain.NewReader(buffer.Bytes())); err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(*blockWitness, c.witness) {
				t.Errorf("test read block commitment fail, got:%v, want:%v", *blockWitness, c.witness)
			}
		})
	}
}

func TestBlockWitnessSet(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want BlockWitness
	}{
		{
			name: "shorter than normal block witness length",
			data: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c12"),
			want: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c120000"),
		},
		{
			name: "longer than normal block witness length",
			data: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c1277091111"),
			want: testutil.MustDecodeHexString("9dc8892df991e1d1110a5cb1bbfd57f2f5e3aa89464de50f9555c7575d9c2b21cf8f05b77b880d8ae4dd218efb15b775c32c9d77f9a2955d69dca9020c127709"),
		},
		{
			name: "normal block witness",
			data: testutil.MustDecodeHexString("2c27ea6e848a1191f25a7f4a04deae1c5a191587e5ee61f92e408ab97dbd35c3ce613b08475f0baa300606c38695d1eb0c4b409939acaa28b82fbb87e7de3c0f"),
			want: testutil.MustDecodeHexString("2c27ea6e848a1191f25a7f4a04deae1c5a191587e5ee61f92e408ab97dbd35c3ce613b08475f0baa300606c38695d1eb0c4b409939acaa28b82fbb87e7de3c0f"),
		},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			witness := &BlockWitness{}
			witness.Set(c.data)
			if !testutil.DeepEqual(c.want, *witness) {
				t.Errorf("update result mismatch: %v, got:%v, want:%v", i, witness, c.want)
			}
		})
	}
}
