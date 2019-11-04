package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/bytom/encoding/blockchain"
	"github.com/bytom/bytom/testutil"
)

func TestReadWriteBlockCommitment(t *testing.T) {
	cases := []struct {
		bc        BlockCommitment
		hexString string
	}{
		{
			bc: BlockCommitment{
				TransactionsMerkleRoot: testutil.MustDecodeHash("35a2d11158f47a5c5267630b2b6cf9e9a5f79a598085a2572a68defeb8013ad2"),
				TransactionStatusHash:  testutil.MustDecodeHash("6978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a"),
			},
			hexString: "35a2d11158f47a5c5267630b2b6cf9e9a5f79a598085a2572a68defeb8013ad26978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a",
		},
		{
			bc: BlockCommitment{
				TransactionsMerkleRoot: testutil.MustDecodeHash("8ec3ee7589f95eee9b534f71fcd37142bcc839a0dbfe78124df9663827b90c35"),
				TransactionStatusHash:  testutil.MustDecodeHash("011bd3380852b2946df507e0c6234222c559eec8f545e4bc58a89e960892259b"),
			},
			hexString: "8ec3ee7589f95eee9b534f71fcd37142bcc839a0dbfe78124df9663827b90c35011bd3380852b2946df507e0c6234222c559eec8f545e4bc58a89e960892259b",
		},
	}

	for _, c := range cases {
		buff := []byte{}
		buffer := bytes.NewBuffer(buff)
		if err := c.bc.writeTo(buffer); err != nil {
			t.Fatal(err)
		}

		hexString := hex.EncodeToString(buffer.Bytes())
		if hexString != c.hexString {
			t.Errorf("test write block commitment fail, got:%s, want:%s", hexString, c.hexString)
		}

		bc := &BlockCommitment{}
		if err := bc.readFrom(blockchain.NewReader(buffer.Bytes())); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(*bc, c.bc) {
			t.Errorf("test read block commitment fail, got:%v, want:%v", *bc, c.bc)
		}
	}
}
