package validation

import (
	"math"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/consensus"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/bytom/testutil"
)

func init() {
	spew.Config.DisableMethods = true
}

func TestGasStatus(t *testing.T) {
	cases := []struct {
		input  *GasState
		output *GasState
		f      func(*GasState) error
		err    error
	}{
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  10000 / consensus.VMGasRate,
				GasUsed:  0,
				BTMValue: 10000,
			},
			f: func(input *GasState) error {
				return input.setGas(10000, 0)
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.setGas(-10000, 0)
			},
			err: errGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:  consensus.DefaultGasCredit,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  100000,
				GasUsed:  0,
				BTMValue: 80000000000,
			},
			f: func(input *GasState) error {
				return input.setGas(80000000000, 0)
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.updateUsage(-1)
			},
			err: errGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  9999,
				GasUsed:  1,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.updateUsage(9999)
			},
			err: nil,
		},
	}

	for i, c := range cases {
		err := c.f(c.input)

		if rootErr(err) != c.err {
			t.Errorf("case %d: got error %s, want %s", i, err, c.err)
		} else if *c.input != *c.output {
			t.Errorf("case %d: gasStatus %v, want %v;", i, c.input, c.output)
		}
	}
}

func TestTxValidation(t *testing.T) {
	var (
		tx      *bc.Tx
		vs      *validationState
		fixture *txFixture

		// the mux from tx, pulled out for convenience
		mux *bc.Mux
	)

	cases := []struct {
		desc string // description of the test case
		f    func() // function to adjust tx, vs, and/or mux
		err  error  // expected error
	}{
		{
			desc: "base case",
		},
		{
			desc: "unbalanced mux amounts",
			f: func() {
				mux.Sources[0].Value.Amount++
				iss := tx.Entries[*mux.Sources[0].Ref].(*bc.Issuance)
				iss.WitnessDestination.Value.Amount++
			},
			err: errUnbalanced,
		},
		{
			desc: "overflowing mux source amounts",
			f: func() {
				mux.Sources[0].Value.Amount = math.MaxInt64
				iss := tx.Entries[*mux.Sources[0].Ref].(*bc.Issuance)
				iss.WitnessDestination.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "underflowing mux destination amounts",
			f: func() {
				mux.WitnessDestinations[0].Value.Amount = math.MaxInt64
				out := tx.Entries[*mux.WitnessDestinations[0].Ref].(*bc.Output)
				out.Source.Value.Amount = math.MaxInt64
				mux.WitnessDestinations[1].Value.Amount = math.MaxInt64
				out = tx.Entries[*mux.WitnessDestinations[1].Ref].(*bc.Output)
				out.Source.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "unbalanced mux assets",
			f: func() {
				mux.Sources[1].Value.AssetId = newAssetID(255)
				sp := tx.Entries[*mux.Sources[1].Ref].(*bc.Spend)
				sp.WitnessDestination.Value.AssetId = newAssetID(255)
			},
			err: errUnbalanced,
		},
		{
			desc: "mismatched output source / mux dest position",
			f: func() {
				tx.Entries[*tx.ResultIds[0]].(*bc.Output).Source.Position = 1
			},
			err: errMismatchedPosition,
		},
		{
			desc: "mismatched output source and mux dest",
			f: func() {
				// For this test, it's necessary to construct a mostly
				// identical second transaction in order to get a similar but
				// not equal output entry for the mux to falsely point
				// to. That entry must be added to the first tx's Entries map.
				fixture2 := sample(t, fixture)
				tx2 := types.NewTx(*fixture2.tx).Tx
				out2ID := tx2.ResultIds[0]
				out2 := tx2.Entries[*out2ID].(*bc.Output)
				tx.Entries[*out2ID] = out2
				mux.WitnessDestinations[0].Ref = out2ID
			},
			err: errMismatchedReference,
		},
		{
			desc: "invalid mux destination position",
			f: func() {
				mux.WitnessDestinations[0].Position = 1
			},
			err: errPosition,
		},
		{
			desc: "mismatched mux dest value / output source value",
			f: func() {
				outID := tx.ResultIds[0]
				out := tx.Entries[*outID].(*bc.Output)
				mux.WitnessDestinations[0].Value = &bc.AssetAmount{
					AssetId: out.Source.Value.AssetId,
					Amount:  out.Source.Value.Amount + 1,
				}
				mux.Sources[0].Value.Amount++ // the mux must still balance
			},
			err: errMismatchedValue,
		},
		{
			desc: "empty tx results",
			f: func() {
				tx.ResultIds = nil
			},
			err: errEmptyResults,
		},
		{
			desc: "empty tx results, but that's OK",
			f: func() {
				tx.Version = 2
				tx.ResultIds = nil
			},
		},
		{
			desc: "issuance program failure",
			f: func() {
				iss := txIssuance(t, tx, 0)
				iss.WitnessArguments[0] = []byte{}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "spend control program failure",
			f: func() {
				spend := txSpend(t, tx, 1)
				spend.WitnessArguments[0] = []byte{}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "mismatched spent source/witness value",
			f: func() {
				spend := txSpend(t, tx, 1)
				spentOutput := tx.Entries[*spend.SpentOutputId].(*bc.Output)
				spentOutput.Source.Value = &bc.AssetAmount{
					AssetId: spend.WitnessDestination.Value.AssetId,
					Amount:  spend.WitnessDestination.Value.Amount + 1,
				}
			},
			err: errMismatchedValue,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			fixture = sample(t, nil)
			tx = types.NewTx(*fixture.tx).Tx
			vs = &validationState{
				block:   mockBlock(),
				tx:      tx,
				entryID: tx.ID,
				gasStatus: &GasState{
					GasLeft: int64(80000),
					GasUsed: 0,
				},
				cache: make(map[bc.Hash]error),
			}
			out := tx.Entries[*tx.ResultIds[0]].(*bc.Output)
			muxID := out.Source.Ref
			mux = tx.Entries[*muxID].(*bc.Mux)

			if c.f != nil {
				c.f()
			}
			err := checkValid(vs, tx.TxHeader)

			if rootErr(err) != c.err {
				t.Errorf("got error %s, want %s; validationState is:\n%s", err, c.err, spew.Sdump(vs))
			}
		})
	}
}

func TestCoinbase(t *testing.T) {
	CbTx := mockCoinbaseTx(5000000000)
	cases := []struct {
		block    *bc.Block
		tx       *bc.Tx
		GasVaild bool
		err      error
	}{
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: 666,
				},
				Transactions: []*bc.Tx{CbTx},
			},
			tx:       CbTx,
			GasVaild: true,
			err:      nil,
		},
	}

	for i, c := range cases {
		gasStatus, err := ValidateTx(c.tx, c.block)

		if rootErr(err) != c.err {
			t.Errorf("#%d got error %s, want %s", i, err, c.err)
		}
		if c.GasVaild != gasStatus.GasVaild {
			t.Errorf("#%d got GasVaild %t, want %t", i, gasStatus.GasVaild, c.GasVaild)
		}
	}
}

func TestTimeRange(t *testing.T) {
	cases := []struct {
		timeRange uint64
		err       bool
	}{
		{
			timeRange: 0,
			err:       false,
		},
		{
			timeRange: 334,
			err:       false,
		},
		{
			timeRange: 332,
			err:       true,
		},
		{
			timeRange: 1521625824,
			err:       false,
		},
		{
			timeRange: 1421625824,
			err:       true,
		},
	}

	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:    333,
			Timestamp: 1521625823,
		},
	}

	tx := types.MapTx(&types.TxData{
		SerializedSize: 1,
		TimeRange:      0,
		Inputs: []*types.TxInput{
			mockGasTxInput(),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, 1, []byte{0x6a}),
		},
	})

	for i, c := range cases {
		tx.TimeRange = c.timeRange
		if _, err := ValidateTx(tx, block); (err != nil) != c.err {
			t.Errorf("#%d got error %t, want %t", i, !c.err, c.err)
		}
	}
}

// A txFixture is returned by sample (below) to produce a sample
// transaction, which takes a separate, optional _input_ txFixture to
// affect the transaction that's built. The components of the
// transaction are the fields of txFixture.
type txFixture struct {
	initialBlockID bc.Hash
	issuanceProg   bc.Program
	issuanceArgs   [][]byte
	assetDef       []byte
	assetID        bc.AssetID
	txVersion      uint64
	txInputs       []*types.TxInput
	txOutputs      []*types.TxOutput
	txRefData      []byte
	tx             *types.TxData
}

// Produces a sample transaction in a txFixture object (see above). A
// separate input txFixture can be used to alter the transaction
// that's created.
//
// The output of this function can be used as the input to a
// subsequent call to make iterative refinements to a test object.
//
// The default transaction produced is valid and has three inputs:
//  - an issuance of 10 units
//  - a spend of 20 units
//  - a spend of 40 units
// and two outputs, one of 25 units and one of 45 units.
// All amounts are denominated in the same asset.
//
// The issuance program for the asset requires two numbers as
// arguments that add up to 5. The prevout control programs require
// two numbers each, adding to 9 and 13, respectively.
//
// The min and max times for the transaction are now +/- one minute.
func sample(tb testing.TB, in *txFixture) *txFixture {
	var result txFixture
	if in != nil {
		result = *in
	}

	if result.initialBlockID.IsZero() {
		result.initialBlockID = *newHash(1)
	}
	if testutil.DeepEqual(result.issuanceProg, bc.Program{}) {
		prog, err := vm.Assemble("ADD 5 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		result.issuanceProg = bc.Program{VmVersion: 1, Code: prog}
	}
	if len(result.issuanceArgs) == 0 {
		result.issuanceArgs = [][]byte{[]byte{2}, []byte{3}}
	}
	if len(result.assetDef) == 0 {
		result.assetDef = []byte{2}
	}
	if result.assetID.IsZero() {
		refdatahash := hashData(result.assetDef)
		result.assetID = bc.ComputeAssetID(result.issuanceProg.Code, result.issuanceProg.VmVersion, &refdatahash)
	}

	if result.txVersion == 0 {
		result.txVersion = 1
	}
	if len(result.txInputs) == 0 {
		cp1, err := vm.Assemble("ADD 9 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args1 := [][]byte{[]byte{4}, []byte{5}}

		cp2, err := vm.Assemble("ADD 13 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args2 := [][]byte{[]byte{6}, []byte{7}}

		result.txInputs = []*types.TxInput{
			types.NewIssuanceInput([]byte{3}, 10, result.issuanceProg.Code, result.issuanceArgs, result.assetDef),
			types.NewSpendInput(args1, *newHash(5), result.assetID, 20, 0, cp1),
			types.NewSpendInput(args2, *newHash(8), result.assetID, 40, 0, cp2),
		}
	}

	result.txInputs = append(result.txInputs, mockGasTxInput())

	if len(result.txOutputs) == 0 {
		cp1, err := vm.Assemble("ADD 17 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		cp2, err := vm.Assemble("ADD 21 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}

		result.txOutputs = []*types.TxOutput{
			types.NewTxOutput(result.assetID, 25, cp1),
			types.NewTxOutput(result.assetID, 45, cp2),
		}
	}
	if len(result.txRefData) == 0 {
		result.txRefData = []byte{13}
	}

	result.tx = &types.TxData{
		Version: result.txVersion,
		Inputs:  result.txInputs,
		Outputs: result.txOutputs,
	}

	return &result
}

func mockBlock() *bc.Block {
	return &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height: 666,
		},
	}
}

func mockCoinbaseTx(amount uint64) *bc.Tx {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	return types.MapTx(&types.TxData{
		SerializedSize: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput(nil),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, amount, cp),
		},
	})
}

func mockGasTxInput() *types.TxInput {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	return types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)
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

func hashData(data []byte) bc.Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], data)
	return bc.NewHash(b32)
}

func newHash(n byte) *bc.Hash {
	h := bc.NewHash([32]byte{n})
	return &h
}

func newAssetID(n byte) *bc.AssetID {
	a := bc.NewAssetID([32]byte{n})
	return &a
}

func txIssuance(t *testing.T, tx *bc.Tx, index int) *bc.Issuance {
	id := tx.InputIDs[index]
	res, err := tx.Issuance(id)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func txSpend(t *testing.T, tx *bc.Tx, index int) *bc.Spend {
	id := tx.InputIDs[index]
	res, err := tx.Spend(id)
	if err != nil {
		t.Fatal(err)
	}
	return res
}
