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
		witness   BlockWitness
		hexString string
	}{
		{
			witness:   BlockWitness{0xbe, 0xef},
			hexString: "02beef",
		},
		{
			witness:   BlockWitness{0xab, 0xcd},
			hexString: "02abcd",
		},
		{
			witness:   BlockWitness{0xcd, 0x68},
			hexString: "02cd68",
		},
		{
			witness:   BlockWitness{},
			hexString: "00",
		},
	}

	for _, c := range cases {
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
	}
}

func TestBlockWitnessSet(t *testing.T) {
	signatureLength := 4
	cases := []struct {
		bw   BlockWitness
		data []byte
		want BlockWitness
	}{
		{
			bw:   make(BlockWitness, signatureLength),
			data: []byte{0x01, 0x02, 0x03, 0x04},
			want: BlockWitness{0x01, 0x02, 0x03, 0x04},
		},
		{
			bw:   BlockWitness{0x01, 0x02, 0x03, 0x04},
			data: []byte{0x01, 0x01, 0x01, 0x01},
			want: BlockWitness{0x01, 0x01, 0x01, 0x01},
		},
	}
	for i, c := range cases {
		newbw := c.bw
		newbw.Set(c.data)
		if !testutil.DeepEqual(c.want, newbw) {
			t.Errorf("update result mismatch: %v, got:%v, want:%v", i, newbw, c.want)
		}
	}
}
