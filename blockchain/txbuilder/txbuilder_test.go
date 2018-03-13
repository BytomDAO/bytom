package txbuilder

import (
	"context"
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/crypto/sha3"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
)

type testAction bc.AssetAmount

func (t testAction) Build(ctx context.Context, b *TemplateBuilder) error {
	in := legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *t.AssetId, t.Amount, 0, nil, bc.Hash{}, nil)
	tplIn := &SigningInstruction{}

	err := b.AddInput(in, tplIn)
	if err != nil {
		return err
	}
	return b.AddOutput(legacy.NewTxOutput(*t.AssetId, t.Amount, []byte("change"), nil))
}

func newControlProgramAction(assetAmt bc.AssetAmount, script []byte) *controlProgramAction {
	return &controlProgramAction{
		AssetAmount: assetAmt,
		Program:     script,
	}
}

func TestBuild(t *testing.T) {
	ctx := context.Background()

	assetID1 := bc.NewAssetID([32]byte{1})
	assetID2 := bc.NewAssetID([32]byte{2})

	actions := []Action{
		newControlProgramAction(bc.AssetAmount{AssetId: &assetID2, Amount: 6}, []byte("dest")),
		testAction(bc.AssetAmount{AssetId: &assetID1, Amount: 5}),
	}
	expiryTime := time.Now().Add(time.Minute)
	got, err := Build(ctx, nil, actions, expiryTime)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := &Template{
		Transaction: legacy.NewTx(legacy.TxData{
			Version:        1,
			SerializedSize: 402,
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), assetID1, 5, 0, nil, bc.Hash{}, nil),
			},
			Outputs: []*legacy.TxOutput{
				legacy.NewTxOutput(assetID2, 6, []byte("dest"), nil),
				legacy.NewTxOutput(assetID1, 5, []byte("change"), nil),
			},
		}),
		SigningInstructions: []*SigningInstruction{{
			WitnessComponents: []witnessComponent{},
		}},
	}

	if !testutil.DeepEqual(got.Transaction.TxData, want.Transaction.TxData) {
		t.Errorf("got tx:\n%s\nwant tx:\n%s", spew.Sdump(got.Transaction.TxData), spew.Sdump(want.Transaction.TxData))
	}

	if !testutil.DeepEqual(got.SigningInstructions, want.SigningInstructions) {
		t.Errorf("got signing instructions:\n\t%#v\nwant signing instructions:\n\t%#v", got.SigningInstructions, want.SigningInstructions)
	}
}

func TestSignatureWitnessMaterialize(t *testing.T) {
	var initialBlockHash bc.Hash
	privkey1, pubkey1, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	privkey2, pubkey2, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	privkey3, pubkey3, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	issuanceProg, _ := vmutil.P2SPMultiSigProgram([]ed25519.PublicKey{pubkey1.PublicKey(), pubkey2.PublicKey(), pubkey3.PublicKey()}, 2)
	assetID := bc.ComputeAssetID(issuanceProg, &initialBlockHash, 1, &bc.EmptyStringHash)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := legacy.NewTx(legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			legacy.NewIssuanceInput([]byte{1}, 100, nil, initialBlockHash, issuanceProg, nil, nil),
		},
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(assetID, 100, outscript, nil),
		},
	})

	tpl := &Template{
		Transaction: unsigned,
	}
	h := tpl.Hash(0)
	builder := vmutil.NewBuilder()
	builder.AddData(h.Bytes())
	builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
	prog, _ := builder.Build()
	msg := sha3.Sum256(prog)
	sig1 := privkey1.Sign(msg[:])
	sig2 := privkey2.Sign(msg[:])
	sig3 := privkey3.Sign(msg[:])
	want := [][]byte{
		vm.Int64Bytes(0),
		sig1,
		sig2,
		prog,
	}

	// Test with more signatures than required, in correct order
	tpl.SigningInstructions = []*SigningInstruction{{
		WitnessComponents: []witnessComponent{
			&SignatureWitness{
				Quorum: 2,
				Keys: []keyID{
					{
						XPub:           pubkey1,
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey2,
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey3,
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
				},
				Program: prog,
				Sigs:    []json.HexBytes{sig1, sig2, sig3},
			},
		},
	}}
	err = materializeWitnesses(tpl)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	got := tpl.Transaction.Inputs[0].Arguments()
	if !testutil.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with exact amount of signatures required, in correct order
	component := tpl.SigningInstructions[0].WitnessComponents[0].(*SignatureWitness)
	component.Sigs = []json.HexBytes{sig1, sig2}
	err = materializeWitnesses(tpl)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	got = tpl.Transaction.Inputs[0].Arguments()
	if !testutil.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}
}

func mustDecodeHex(str string) []byte {
	data, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return data
}

func TestCheckBlankCheck(t *testing.T) {
	cases := []struct {
		tx   *legacy.TxData
		want error
	}{{
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &legacy.TxData{
			Inputs:  []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 3, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil),
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.NewAssetID([32]byte{1}), 5, 0, nil, bc.Hash{}, nil),
			},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 5, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{
				legacy.NewTxOutput(bc.AssetID{}, math.MaxInt64, nil, nil),
				legacy.NewTxOutput(bc.AssetID{}, 7, nil, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil),
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, math.MaxInt64, 0, nil, bc.Hash{}, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &legacy.TxData{
			Inputs:  []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 5, nil, nil)},
		},
		want: nil,
	}, {
		tx: &legacy.TxData{
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 5, nil, nil)},
		},
		want: nil,
	}, {
		tx: &legacy.TxData{
			Inputs:  []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.NewAssetID([32]byte{1}), 5, nil, nil)},
		},
		want: nil,
	}}

	for _, c := range cases {
		got := checkBlankCheck(c.tx)
		if errors.Root(got) != c.want {
			t.Errorf("checkUnsafe(%+v) err = %v want %v", c.tx, errors.Root(got), c.want)
		}
	}
}
