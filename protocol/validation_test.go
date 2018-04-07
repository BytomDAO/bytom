package protocol

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
)

func init() {
	spew.Config.DisableMethods = true
}

func TestValidateBlock(t *testing.T) {
	cases := []struct {
		block *bc.Block
		err   error
	}{
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:          0,
					Bits:            2305843009230471167,
					PreviousBlockId: &bc.Hash{},
				},
				Transactions: []*bc.Tx{mockCoinbaseTx(23, 1470000000000000000).Tx},
			},
			err: nil,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:          0,
					Bits:            2305843009230471167,
					PreviousBlockId: &bc.Hash{},
				},
				Transactions: []*bc.Tx{mockCoinbaseTx(23, 1).Tx},
			},
			err: errWrongCoinbaseTransaction,
		},
	}

	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, err := bc.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		t.Error(err)
	}

	for _, c := range cases {
		txRoot, err := bc.TxMerkleRoot(c.block.Transactions)
		if err != nil {
			t.Errorf("computing transaction merkle root error: %v", err)
			continue
		}

		c.block.BlockHeader.TransactionStatus = bc.NewTransactionStatus()
		c.block.TransactionsRoot = &txRoot
		c.block.TransactionStatusHash = &txStatusHash

		if err = ValidateBlock(c.block, nil, &bc.Hash{}, nil); rootErr(err) != c.err {
			t.Errorf("got error %s, want %s", err, c.err)
		}
	}
}

func TestBlockHeaderValid(t *testing.T) {
	base := bc.NewBlockHeader(1, 1, &bc.Hash{}, 1, &bc.Hash{}, &bc.Hash{}, 0, 0)
	baseBytes, _ := proto.Marshal(base)

	var bh bc.BlockHeader

	cases := []struct {
		f   func()
		err error
	}{
		{},
		{
			f: func() {
				bh.Version = 2
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			proto.Unmarshal(baseBytes, &bh)
			if c.f != nil {
				c.f()
			}
		})
	}
}

// Like errors.Root, but also unwraps vm.Error objects.
func rootErr(e error) error {
	for {
		e = errors.Root(e)
		if e2, ok := e.(vm.Error); ok {
			e = e2.Err
			continue
		}
		return e
	}
}
