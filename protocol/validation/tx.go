package validation

import (
	"fmt"
	"math"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/errors"
	"github.com/bytom/math/checked"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
)

const ruleAA = 142500

// validate transaction error
var (
	ErrTxVersion                 = errors.New("invalid transaction version")
	ErrWrongTransactionSize      = errors.New("invalid transaction size")
	ErrBadTimeRange              = errors.New("invalid transaction time range")
	ErrEmptyInputIDs             = errors.New("got the empty InputIDs")
	ErrNotStandardTx             = errors.New("not standard transaction")
	ErrWrongCoinbaseTransaction  = errors.New("wrong coinbase transaction")
	ErrWrongCoinbaseAsset        = errors.New("wrong coinbase assetID")
	ErrCoinbaseArbitraryOversize = errors.New("coinbase arbitrary size is larger than limit")
	ErrEmptyResults              = errors.New("transaction has no results")
	ErrMismatchedAssetID         = errors.New("mismatched assetID")
	ErrMismatchedPosition        = errors.New("mismatched value source/dest position")
	ErrMismatchedReference       = errors.New("mismatched reference")
	ErrMismatchedValue           = errors.New("mismatched value")
	ErrMissingField              = errors.New("missing required field")
	ErrNoSource                  = errors.New("no source for value")
	ErrOverflow                  = errors.New("arithmetic overflow/underflow")
	ErrPosition                  = errors.New("invalid source or destination position")
	ErrUnbalanced                = errors.New("unbalanced asset amount between input and output")
	ErrOverGasCredit             = errors.New("all gas credit has been spend")
	ErrGasCalculate              = errors.New("gas usage calculate got a math error")
)

// GasState record the gas usage status
type GasState struct {
	BTMValue   uint64
	GasLeft    int64
	GasUsed    int64
	GasValid   bool
	StorageGas int64
}

func (g *GasState) setGas(BTMValue int64, txSize int64) error {
	if BTMValue < 0 {
		return errors.Wrap(ErrGasCalculate, "input BTM is negative")
	}

	g.BTMValue = uint64(BTMValue)

	var ok bool
	if g.GasLeft, ok = checked.DivInt64(BTMValue, consensus.VMGasRate); !ok {
		return errors.Wrap(ErrGasCalculate, "setGas calc gas amount")
	}

	if g.GasLeft > consensus.MaxGasAmount {
		g.GasLeft = consensus.MaxGasAmount
	}

	if g.StorageGas, ok = checked.MulInt64(txSize, consensus.StorageGasRate); !ok {
		return errors.Wrap(ErrGasCalculate, "setGas calc tx storage gas")
	}
	return nil
}

func (g *GasState) setGasValid() error {
	var ok bool
	if g.GasLeft, ok = checked.SubInt64(g.GasLeft, g.StorageGas); !ok || g.GasLeft < 0 {
		return errors.Wrap(ErrGasCalculate, "setGasValid calc gasLeft")
	}

	if g.GasUsed, ok = checked.AddInt64(g.GasUsed, g.StorageGas); !ok {
		return errors.Wrap(ErrGasCalculate, "setGasValid calc gasUsed")
	}

	g.GasValid = true
	return nil
}

func (g *GasState) updateUsage(gasLeft int64) error {
	if gasLeft < 0 {
		return errors.Wrap(ErrGasCalculate, "updateUsage input negative gas")
	}

	if gasUsed, ok := checked.SubInt64(g.GasLeft, gasLeft); ok {
		g.GasUsed += gasUsed
		g.GasLeft = gasLeft
	} else {
		return errors.Wrap(ErrGasCalculate, "updateUsage calc gas diff")
	}

	if !g.GasValid && (g.GasUsed > consensus.DefaultGasCredit || g.StorageGas > g.GasLeft) {
		return ErrOverGasCredit
	}
	return nil
}

// validationState contains the context that must propagate through
// the transaction graph when validating entries.
type validationState struct {
	block     *bc.Block
	tx        *bc.Tx
	gasStatus *GasState
	entryID   bc.Hash           // The ID of the nearest enclosing entry
	sourcePos uint64            // The source position, for validate ValueSources
	destPos   uint64            // The destination position, for validate ValueDestinations
	cache     map[bc.Hash]error // Memoized per-entry validation results
}

func checkValid(vs *validationState, e bc.Entry) (err error) {
	var ok bool
	entryID := bc.EntryID(e)
	if err, ok = vs.cache[entryID]; ok {
		return err
	}

	defer func() {
		vs.cache[entryID] = err
	}()

	switch e := e.(type) {
	case *bc.TxHeader:
		for i, resID := range e.ResultIds {
			resultEntry := vs.tx.Entries[*resID]
			vs2 := *vs
			vs2.entryID = *resID
			if err = checkValid(&vs2, resultEntry); err != nil {
				return errors.Wrapf(err, "checking result %d", i)
			}
		}

		if e.Version == 1 && len(e.ResultIds) == 0 {
			return ErrEmptyResults
		}

	case *bc.Mux:
		parity := make(map[bc.AssetID]int64)
		for i, src := range e.Sources {
			if src.Value.Amount > math.MaxInt64 {
				return errors.WithDetailf(ErrOverflow, "amount %d exceeds maximum value 2^63", src.Value.Amount)
			}
			sum, ok := checked.AddInt64(parity[*src.Value.AssetId], int64(src.Value.Amount))
			if !ok {
				return errors.WithDetailf(ErrOverflow, "adding %d units of asset %x from mux source %d to total %d overflows int64", src.Value.Amount, src.Value.AssetId.Bytes(), i, parity[*src.Value.AssetId])
			}
			parity[*src.Value.AssetId] = sum
		}

		for i, dest := range e.WitnessDestinations {
			sum, ok := parity[*dest.Value.AssetId]
			if !ok {
				return errors.WithDetailf(ErrNoSource, "mux destination %d, asset %x, has no corresponding source", i, dest.Value.AssetId.Bytes())
			}
			if dest.Value.Amount > math.MaxInt64 {
				return errors.WithDetailf(ErrOverflow, "amount %d exceeds maximum value 2^63", dest.Value.Amount)
			}
			diff, ok := checked.SubInt64(sum, int64(dest.Value.Amount))
			if !ok {
				return errors.WithDetailf(ErrOverflow, "subtracting %d units of asset %x from mux destination %d from total %d underflows int64", dest.Value.Amount, dest.Value.AssetId.Bytes(), i, sum)
			}
			parity[*dest.Value.AssetId] = diff
		}

		for assetID, amount := range parity {
			if assetID == *consensus.BTMAssetID {
				if err = vs.gasStatus.setGas(amount, int64(vs.tx.SerializedSize)); err != nil {
					return err
				}
			} else if amount != 0 {
				return errors.WithDetailf(ErrUnbalanced, "asset %x sources - destinations = %d (should be 0)", assetID.Bytes(), amount)
			}
		}

		for _, BTMInputID := range vs.tx.GasInputIDs {
			e, ok := vs.tx.Entries[BTMInputID]
			if !ok {
				return errors.Wrapf(bc.ErrMissingEntry, "entry for bytom input %x not found", BTMInputID)
			}

			vs2 := *vs
			vs2.entryID = BTMInputID
			if err := checkValid(&vs2, e); err != nil {
				return errors.Wrap(err, "checking gas input")
			}
		}

		for i, dest := range e.WitnessDestinations {
			vs2 := *vs
			vs2.destPos = uint64(i)
			if err = checkValidDest(&vs2, dest); err != nil {
				return errors.Wrapf(err, "checking mux destination %d", i)
			}
		}

		if err := vs.gasStatus.setGasValid(); err != nil {
			return err
		}

		for i, src := range e.Sources {
			vs2 := *vs
			vs2.sourcePos = uint64(i)
			if err = checkValidSrc(&vs2, src); err != nil {
				return errors.Wrapf(err, "checking mux source %d", i)
			}
		}

	case *bc.Output:
		vs2 := *vs
		vs2.sourcePos = 0
		if err = checkValidSrc(&vs2, e.Source); err != nil {
			return errors.Wrap(err, "checking output source")
		}

	case *bc.Retirement:
		vs2 := *vs
		vs2.sourcePos = 0
		if err = checkValidSrc(&vs2, e.Source); err != nil {
			return errors.Wrap(err, "checking retirement source")
		}

	case *bc.Issuance:
		computedAssetID := e.WitnessAssetDefinition.ComputeAssetID()
		if computedAssetID != *e.Value.AssetId {
			return errors.WithDetailf(ErrMismatchedAssetID, "asset ID is %x, issuance wants %x", computedAssetID.Bytes(), e.Value.AssetId.Bytes())
		}

		gasLeft, err := vm.Verify(NewTxVMContext(vs, e, e.WitnessAssetDefinition.IssuanceProgram, e.WitnessArguments), vs.gasStatus.GasLeft)
		if err != nil {
			return errors.Wrap(err, "checking issuance program")
		}
		if err = vs.gasStatus.updateUsage(gasLeft); err != nil {
			return err
		}

		destVS := *vs
		destVS.destPos = 0
		if err = checkValidDest(&destVS, e.WitnessDestination); err != nil {
			return errors.Wrap(err, "checking issuance destination")
		}

	case *bc.Spend:
		if e.SpentOutputId == nil {
			return errors.Wrap(ErrMissingField, "spend without spent output ID")
		}
		spentOutput, err := vs.tx.Output(*e.SpentOutputId)
		if err != nil {
			return errors.Wrap(err, "getting spend prevout")
		}

		gasLeft, err := vm.Verify(NewTxVMContext(vs, e, spentOutput.ControlProgram, e.WitnessArguments), vs.gasStatus.GasLeft)
		if err != nil {
			return errors.Wrap(err, "checking control program")
		}
		if err = vs.gasStatus.updateUsage(gasLeft); err != nil {
			return err
		}

		eq, err := spentOutput.Source.Value.Equal(e.WitnessDestination.Value)
		if err != nil {
			return err
		}
		if !eq {
			return errors.WithDetailf(
				ErrMismatchedValue,
				"previous output is for %d unit(s) of %x, spend wants %d unit(s) of %x",
				spentOutput.Source.Value.Amount,
				spentOutput.Source.Value.AssetId.Bytes(),
				e.WitnessDestination.Value.Amount,
				e.WitnessDestination.Value.AssetId.Bytes(),
			)
		}

		vs2 := *vs
		vs2.destPos = 0
		if err = checkValidDest(&vs2, e.WitnessDestination); err != nil {
			return errors.Wrap(err, "checking spend destination")
		}

	case *bc.Coinbase:
		if vs.block == nil || len(vs.block.Transactions) == 0 || vs.block.Transactions[0] != vs.tx {
			return ErrWrongCoinbaseTransaction
		}

		if *e.WitnessDestination.Value.AssetId != *consensus.BTMAssetID {
			return ErrWrongCoinbaseAsset
		}

		if e.Arbitrary != nil && len(e.Arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
			return ErrCoinbaseArbitraryOversize
		}

		vs2 := *vs
		vs2.destPos = 0
		if err = checkValidDest(&vs2, e.WitnessDestination); err != nil {
			return errors.Wrap(err, "checking coinbase destination")
		}
		vs.gasStatus.StorageGas = 0

	default:
		return fmt.Errorf("entry has unexpected type %T", e)
	}

	return nil
}

func checkValidSrc(vstate *validationState, vs *bc.ValueSource) error {
	if vs == nil {
		return errors.Wrap(ErrMissingField, "empty value source")
	}
	if vs.Ref == nil {
		return errors.Wrap(ErrMissingField, "missing ref on value source")
	}
	if vs.Value == nil || vs.Value.AssetId == nil {
		return errors.Wrap(ErrMissingField, "missing value on value source")
	}

	e, ok := vstate.tx.Entries[*vs.Ref]
	if !ok {
		return errors.Wrapf(bc.ErrMissingEntry, "entry for value source %x not found", vs.Ref.Bytes())
	}

	vstate2 := *vstate
	vstate2.entryID = *vs.Ref
	if err := checkValid(&vstate2, e); err != nil {
		return errors.Wrap(err, "checking value source")
	}

	var dest *bc.ValueDestination
	switch ref := e.(type) {
	case *bc.Coinbase:
		if vs.Position != 0 {
			return errors.Wrapf(ErrPosition, "invalid position %d for coinbase source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Issuance:
		if vs.Position != 0 {
			return errors.Wrapf(ErrPosition, "invalid position %d for issuance source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Spend:
		if vs.Position != 0 {
			return errors.Wrapf(ErrPosition, "invalid position %d for spend source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Mux:
		if vs.Position >= uint64(len(ref.WitnessDestinations)) {
			return errors.Wrapf(ErrPosition, "invalid position %d for %d-destination mux source", vs.Position, len(ref.WitnessDestinations))
		}
		dest = ref.WitnessDestinations[vs.Position]

	default:
		return errors.Wrapf(bc.ErrEntryType, "value source is %T, should be coinbase, issuance, spend, or mux", e)
	}

	if dest.Ref == nil || *dest.Ref != vstate.entryID {
		return errors.Wrapf(ErrMismatchedReference, "value source for %x has disagreeing destination %x", vstate.entryID.Bytes(), dest.Ref.Bytes())
	}

	if dest.Position != vstate.sourcePos {
		return errors.Wrapf(ErrMismatchedPosition, "value source position %d disagrees with %d", dest.Position, vstate.sourcePos)
	}

	eq, err := dest.Value.Equal(vs.Value)
	if err != nil {
		return errors.Sub(ErrMissingField, err)
	}
	if !eq {
		return errors.Wrapf(ErrMismatchedValue, "source value %v disagrees with %v", dest.Value, vs.Value)
	}

	return nil
}

func checkValidDest(vs *validationState, vd *bc.ValueDestination) error {
	if vd == nil {
		return errors.Wrap(ErrMissingField, "empty value destination")
	}
	if vd.Ref == nil {
		return errors.Wrap(ErrMissingField, "missing ref on value destination")
	}
	if vd.Value == nil || vd.Value.AssetId == nil {
		return errors.Wrap(ErrMissingField, "missing value on value source")
	}

	e, ok := vs.tx.Entries[*vd.Ref]
	if !ok {
		return errors.Wrapf(bc.ErrMissingEntry, "entry for value destination %x not found", vd.Ref.Bytes())
	}

	var src *bc.ValueSource
	switch ref := e.(type) {
	case *bc.Output:
		if vd.Position != 0 {
			return errors.Wrapf(ErrPosition, "invalid position %d for output destination", vd.Position)
		}
		src = ref.Source

	case *bc.Retirement:
		if vd.Position != 0 {
			return errors.Wrapf(ErrPosition, "invalid position %d for retirement destination", vd.Position)
		}
		src = ref.Source

	case *bc.Mux:
		if vd.Position >= uint64(len(ref.Sources)) {
			return errors.Wrapf(ErrPosition, "invalid position %d for %d-source mux destination", vd.Position, len(ref.Sources))
		}
		src = ref.Sources[vd.Position]

	default:
		return errors.Wrapf(bc.ErrEntryType, "value destination is %T, should be output, retirement, or mux", e)
	}

	if src.Ref == nil || *src.Ref != vs.entryID {
		return errors.Wrapf(ErrMismatchedReference, "value destination for %x has disagreeing source %x", vs.entryID.Bytes(), src.Ref.Bytes())
	}

	if src.Position != vs.destPos {
		return errors.Wrapf(ErrMismatchedPosition, "value destination position %d disagrees with %d", src.Position, vs.destPos)
	}

	eq, err := src.Value.Equal(vd.Value)
	if err != nil {
		return errors.Sub(ErrMissingField, err)
	}
	if !eq {
		return errors.Wrapf(ErrMismatchedValue, "destination value %v disagrees with %v", src.Value, vd.Value)
	}

	return nil
}

func checkStandardTx(tx *bc.Tx, blockHeight uint64) error {
	for _, id := range tx.InputIDs {
		if blockHeight >= ruleAA && id.IsZero() {
			return ErrEmptyInputIDs
		}
	}

	for _, id := range tx.GasInputIDs {
		spend, err := tx.Spend(id)
		if err != nil {
			continue
		}
		spentOutput, err := tx.Output(*spend.SpentOutputId)
		if err != nil {
			return err
		}

		if !segwit.IsP2WScript(spentOutput.ControlProgram.Code) {
			return ErrNotStandardTx
		}
	}

	for _, id := range tx.ResultIds {
		e, ok := tx.Entries[*id]
		if !ok {
			return errors.Wrapf(bc.ErrMissingEntry, "id %x", id.Bytes())
		}

		output, ok := e.(*bc.Output)
		if !ok || *output.Source.Value.AssetId != *consensus.BTMAssetID {
			continue
		}

		if !segwit.IsP2WScript(output.ControlProgram.Code) {
			return ErrNotStandardTx
		}
	}
	return nil
}

func checkTimeRange(tx *bc.Tx, block *bc.Block) error {
	if tx.TimeRange == 0 {
		return nil
	}

	if tx.TimeRange < block.Height {
		return ErrBadTimeRange
	}
	return nil
}

// ValidateTx validates a transaction.
func ValidateTx(tx *bc.Tx, block *bc.Block) (*GasState, error) {
	gasStatus := &GasState{GasValid: false}
	if block.Version == 1 && tx.Version != 1 {
		return gasStatus, errors.WithDetailf(ErrTxVersion, "block version %d, transaction version %d", block.Version, tx.Version)
	}
	if tx.SerializedSize == 0 {
		return gasStatus, ErrWrongTransactionSize
	}
	if err := checkTimeRange(tx, block); err != nil {
		return gasStatus, err
	}
	if err := checkStandardTx(tx, block.Height); err != nil {
		return gasStatus, err
	}

	vs := &validationState{
		block:     block,
		tx:        tx,
		entryID:   tx.ID,
		gasStatus: gasStatus,
		cache:     make(map[bc.Hash]error),
	}
	return vs.gasStatus, checkValid(vs, tx.TxHeader)
}
