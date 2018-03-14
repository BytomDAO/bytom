// Package bctest provides utilities for constructing blockchain data
// structures.
package bctest

import (
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
)

// NewIssuanceTx creates a new signed, issuance transaction issuing 100 units
// of a new asset to a garbage control program. The resulting transaction has
// one input and one output.
//
// The asset issued is created from randomly-generated keys. The resulting
// transaction is finalized (signed with a TXSIGHASH commitment).
func NewIssuanceTx(tb testing.TB, initial bc.Hash, opts ...func(*legacy.Tx)) *legacy.Tx {
	// Generate a random key pair for the asset being issued.
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	pubkeys := chainkd.XPubKeys([]chainkd.XPub{xpub})

	// Create a corresponding issuance program.
	sigProg, err := vmutil.P2SPMultiSigProgram(pubkeys, 1)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	builder := vmutil.NewBuilder()
	builder.AddRawBytes(sigProg)
	issuanceProgram, _ := builder.Build()

	// Create a transaction issuing this new asset.
	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	assetdef := []byte(`{"type": "prottest issuance"}`)
	txin := legacy.NewIssuanceInput(nonce[:], 100, initial, issuanceProgram, nil, assetdef)

	tx := legacy.NewTx(legacy.TxData{
		Version: 1,
		Inputs:  []*legacy.TxInput{txin},
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(txin.AssetID(), 100, []byte{0xbe, 0xef}),
		},
	})

	for _, opt := range opts {
		opt(tx)
	}

	// Sign with a simple TXSIGHASH signature.
	builder = vmutil.NewBuilder()
	h := tx.SigHash(0)
	builder.AddData(h.Bytes())
	builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
	sigprog, _ := builder.Build()
	sigproghash := sha3.Sum256(sigprog)
	signature := xprv.Sign(sigproghash[:])

	var witness [][]byte
	witness = append(witness, vm.Int64Bytes(0)) // 0 args to the sigprog
	witness = append(witness, signature)
	witness = append(witness, sigprog)
	tx.SetInputArguments(0, witness)

	return tx
}
