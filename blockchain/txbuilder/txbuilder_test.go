package txbuilder

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
)

type testAction bc.AssetAmount

func (t testAction) Build(ctx context.Context, b *TemplateBuilder) error {
	in := types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *t.AssetId, t.Amount, 0, nil)
	tplIn := &SigningInstruction{}

	err := b.AddInput(in, tplIn)
	if err != nil {
		return err
	}
	return b.AddOutput(types.NewTxOutput(*t.AssetId, t.Amount, []byte("change")))
}

func (t testAction) ActionType() string {
	return "test-action"
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
	got, err := Build(ctx, nil, actions, expiryTime, 0)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := &Template{
		Transaction: types.NewTx(types.TxData{
			Version: 1,
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), assetID1, 5, 0, nil),
			},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(assetID2, 6, []byte("dest")),
				types.NewTxOutput(assetID1, 5, []byte("change")),
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
	assetID := bc.ComputeAssetID(issuanceProg, 1, &bc.EmptyStringHash)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := types.NewTx(types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewIssuanceInput([]byte{1}, 100, issuanceProg, nil, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(assetID, 100, outscript),
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
						DerivationPath: []chainjson.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey2,
						DerivationPath: []chainjson.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey3,
						DerivationPath: []chainjson.HexBytes{{0, 0, 0, 0}},
					},
				},
				Program: prog,
				Sigs:    []chainjson.HexBytes{sig1, sig2, sig3},
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
	component.Sigs = []chainjson.HexBytes{sig1, sig2}
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
		tx   *types.TxData
		want error
	}{{
		tx: &types.TxData{
			Inputs: []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil)},
			Outputs: []*types.TxOutput{types.NewTxOutput(bc.AssetID{}, 3, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil),
				types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.NewAssetID([32]byte{1}), 5, 0, nil),
			},
			Outputs: []*types.TxOutput{types.NewTxOutput(bc.AssetID{}, 5, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &types.TxData{
			Inputs: []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil)},
			Outputs: []*types.TxOutput{
				types.NewTxOutput(bc.AssetID{}, math.MaxInt64, nil),
				types.NewTxOutput(bc.AssetID{}, 7, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil),
				types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, math.MaxInt64, 0, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil)},
			Outputs: []*types.TxOutput{types.NewTxOutput(bc.AssetID{}, 5, nil)},
		},
		want: nil,
	}, {
		tx: &types.TxData{
			Outputs: []*types.TxOutput{types.NewTxOutput(bc.AssetID{}, 5, nil)},
		},
		want: nil,
	}, {
		tx: &types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil)},
			Outputs: []*types.TxOutput{types.NewTxOutput(bc.NewAssetID([32]byte{1}), 5, nil)},
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

func TestCreateTxByUtxo(t *testing.T) {
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}

	pub := xpub.PublicKey()
	pubHash := crypto.Ripemd160(pub)
	program, err := vmutil.P2WPKHProgram([]byte(pubHash))
	if err != nil {
		t.Fatal(err)
	}

	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		t.Fatal(err)
	}

	muxID := testutil.MustDecodeHash("1e673900965623ec3305cead5a78dfb68a34599f8bc078460f3f202256c3dfa6")
	utxo := struct {
		SourceID       bc.Hash
		AssetID        bc.AssetID
		Amount         uint64
		SourcePos      uint64
		ControlProgram []byte
		Address        string
	}{
		SourceID:       muxID,
		AssetID:        *consensus.BTMAssetID,
		Amount:         20000000000,
		SourcePos:      1,
		ControlProgram: program,
		Address:        address.EncodeAddress(),
	}

	recvProg := mustDecodeHex("00145056532ecd3621c9ce8adde5505c058610b287cf")
	tx := types.NewTx(types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, utxo.SourceID, utxo.AssetID, utxo.Amount, utxo.SourcePos, utxo.ControlProgram),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, 10000000000, recvProg),
		},
	})

	tpl := &Template{
		Transaction:     tx,
		AllowAdditional: false,
	}

	h := tpl.Hash(0).Byte32()
	sig := xprv.Sign(h[:])
	data := []byte(pub)

	// Test with more signatures than required, in correct order
	tpl.SigningInstructions = []*SigningInstruction{{
		WitnessComponents: []witnessComponent{
			&RawTxSigWitness{
				Quorum: 1,
				Sigs:   []chainjson.HexBytes{sig},
			},
			DataWitness(data),
		},
	}}

	if err = materializeWitnesses(tpl); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(tx, tpl.Transaction) {
		t.Errorf("tx:%v result is equal to want:%v", tx, tpl.Transaction)
	}
}

func TestAddContractArgs(t *testing.T) {
	hexXpub, err := hex.DecodeString("ba76bb52574b3f40315f2c01f1818a9072ced56e9d4b68acbef56a4d0077d08e5e34837963e4cdc54eb251aa34aad01e6ae48b140f6a2743fbb0a0abd9cf8aac")
	if err != nil {
		t.Fatal(err)
	}

	var xpub chainkd.XPub
	copy(xpub[:], hexXpub)

	rawTxSig := RawTxSigArgument{RootXPub: xpub, Path: []chainjson.HexBytes{{1, 1, 0, 0, 0, 0, 0, 0, 0}, {1, 0, 0, 0, 0, 0, 0, 0}}}
	rawTxSigMsg, err := json.Marshal(rawTxSig)
	if err != nil {
		t.Fatal(err)
	}

	data := DataArgument{Value: "7468697320697320612074657374"}
	dataMsg, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		arguments  []ContractArgument
		wantResult error
	}{
		{
			arguments: []ContractArgument{
				{
					Type:    "raw_tx_signature",
					RawData: rawTxSigMsg,
				},
				{
					Type:    "data",
					RawData: dataMsg,
				},
			},
			wantResult: nil,
		},
		{
			arguments: []ContractArgument{
				{
					Type:    "data",
					RawData: dataMsg,
				},
				{
					Type:    "raw_tx_signature",
					RawData: rawTxSigMsg,
				},
			},
			wantResult: nil,
		},
		{
			arguments: []ContractArgument{
				{
					Type:    "data",
					RawData: dataMsg,
				},
				{
					Type:    "err_data",
					RawData: rawTxSigMsg,
				},
			},
			wantResult: ErrBadContractArgType,
		},
	}

	sigInst := &SigningInstruction{}
	for _, c := range cases {
		err := AddContractArgs(sigInst, c.arguments)
		if err != c.wantResult {
			t.Fatalf("got result=%v, want result=%v", err, c.wantResult)
		}
	}
}
