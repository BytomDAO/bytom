package validation

import (
	"fmt"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/errors"
	"github.com/bytom/math/checked"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
)

const muxGasCost = int64(10)

// GasState record the gas usage status
type GasState struct {
	BTMValue   uint64
	GasLeft    int64
	GasUsed    int64
	GasVaild   bool
	storageGas int64
}

func (g *GasState) setGas(BTMValue int64, txSize int64) error {
	if BTMValue < 0 {
		return errors.Wrap(errGasCalculate, "input BTM is negative")
	}

	g.BTMValue = uint64(BTMValue)

	if BTMValue == 0 {
		g.GasLeft = muxGasCost
		return nil
	}

	var ok bool
	if g.GasLeft, ok = checked.DivInt64(BTMValue, consensus.VMGasRate); !ok {
		return errors.Wrap(errGasCalculate, "setGas calc gas amount")
	}

	if g.GasLeft > consensus.MaxGasAmount {
		g.GasLeft = consensus.MaxGasAmount
	}

	if g.storageGas, ok = checked.MulInt64(txSize, consensus.StorageGasRate); !ok {
		return errors.Wrap(errGasCalculate, "setGas calc tx storage gas")
	}
	return nil
}

func (g *GasState) setGasVaild() error {
	var ok bool
	if g.GasLeft, ok = checked.SubInt64(g.GasLeft, g.storageGas); !ok || g.GasLeft < 0 {
		return errors.Wrap(errGasCalculate, "setGasVaild calc gasLeft")
	}

	if g.GasUsed, ok = checked.AddInt64(g.GasUsed, g.storageGas); !ok {
		return errors.Wrap(errGasCalculate, "setGasVaild calc gasUsed")
	}

	g.GasVaild = true
	return nil
}

func (g *GasState) updateUsage(gasLeft int64) error {
	if gasLeft < 0 {
		return errors.Wrap(errGasCalculate, "updateUsage input negative gas")
	}

	if gasUsed, ok := checked.SubInt64(g.GasLeft, gasLeft); ok {
		g.GasUsed += gasUsed
		g.GasLeft = gasLeft
	} else {
		return errors.Wrap(errGasCalculate, "updateUsage calc gas diff")
	}

	if !g.GasVaild && (g.GasUsed > consensus.DefaultGasCredit || g.storageGas > g.GasLeft) {
		return errOverGasCredit
	}
	return nil
}

// validationState contains the context that must propagate through
// the transaction graph when validating entries.
type validationState struct {
	// The ID of the blockchain
	block *bc.Block

	// The enclosing transaction object
	tx *bc.Tx

	// The ID of the nearest enclosing entry
	entryID bc.Hash

	// The source position, for validating ValueSources
	sourcePos uint64

	// The destination position, for validating ValueDestinations
	destPos uint64

	// Memoized per-entry validation results
	cache map[bc.Hash]error

	gasStatus *GasState
}

var (
	errBadTimestamp             = errors.New("block timestamp is great than limit")
	errGasCalculate             = errors.New("gas usage calculate got a math error")
	errEmptyResults             = errors.New("transaction has no results")
	errMismatchedAssetID        = errors.New("mismatched asset id")
	errMismatchedBlock          = errors.New("mismatched block")
	errMismatchedMerkleRoot     = errors.New("mismatched merkle root")
	errMismatchedPosition       = errors.New("mismatched value source/dest positions")
	errMismatchedReference      = errors.New("mismatched reference")
	errMismatchedTxStatus       = errors.New("mismatched transaction status")
	errMismatchedValue          = errors.New("mismatched value")
	errMisorderedBlockHeight    = errors.New("misordered block height")
	errMisorderedBlockTime      = errors.New("misordered block time")
	errMissingField             = errors.New("missing required field")
	errNoGas                    = errors.New("no gas input")
	errNoPrevBlock              = errors.New("no previous block")
	errNoSource                 = errors.New("no source for value")
	errNonemptyExtHash          = errors.New("non-empty extension hash")
	errOverflow                 = errors.New("arithmetic overflow/underflow")
	errOverGasCredit            = errors.New("all gas credit has been spend")
	errOverBlockLimit           = errors.New("block's gas is over the limit")
	errPosition                 = errors.New("invalid source or destination position")
	errWorkProof                = errors.New("invalid difficulty proof of work")
	errTxVersion                = errors.New("invalid transaction version")
	errUnbalanced               = errors.New("unbalanced")
	errUntimelyTransaction      = errors.New("block timestamp outside transaction time range")
	errVersionRegression        = errors.New("version regression")
	errWrongBlockSize           = errors.New("block size is too big")
	errWrongTransactionSize     = errors.New("transaction size is not in vaild range")
	errWrongTransactionStatus   = errors.New("transaction status is wrong")
	errWrongCoinbaseTransaction = errors.New("wrong coinbase transaction")
	errWrongCoinbaseAsset       = errors.New("wrong coinbase asset id")
	errNotStandardTx            = errors.New("gas transaction is not standard transaction")
)

func checkValid(vs *validationState, e bc.Entry) (err error) {
	entryID := bc.EntryID(e)
	var ok bool
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
			err = checkValid(&vs2, resultEntry)
			if err != nil {
				return errors.Wrapf(err, "checking result %d", i)
			}
		}

		if e.Version == 1 {
			if len(e.ResultIds) == 0 {
				return errEmptyResults
			}

			if e.ExtHash != nil && !e.ExtHash.IsZero() {
				return errNonemptyExtHash
			}
		}

	case *bc.Coinbase:
		if vs.block == nil || len(vs.block.Transactions) == 0 || vs.block.Transactions[0] != vs.tx {
			return errWrongCoinbaseTransaction
		}

		if *e.WitnessDestination.Value.AssetId != *consensus.BTMAssetID {
			return errWrongCoinbaseAsset
		}

		vs2 := *vs
		vs2.destPos = 0
		err = checkValidDest(&vs2, e.WitnessDestination)
		if err != nil {
			return errors.Wrap(err, "checking coinbase destination")
		}

	case *bc.Mux:
		parity := make(map[bc.AssetID]int64)
		for i, src := range e.Sources {
			sum, ok := checked.AddInt64(parity[*src.Value.AssetId], int64(src.Value.Amount))
			if !ok {
				return errors.WithDetailf(errOverflow, "adding %d units of asset %x from mux source %d to total %d overflows int64", src.Value.Amount, src.Value.AssetId.Bytes(), i, parity[*src.Value.AssetId])
			}
			parity[*src.Value.AssetId] = sum
		}

		for i, dest := range e.WitnessDestinations {
			sum, ok := parity[*dest.Value.AssetId]
			if !ok {
				return errors.WithDetailf(errNoSource, "mux destination %d, asset %x, has no corresponding source", i, dest.Value.AssetId.Bytes())
			}

			diff, ok := checked.SubInt64(sum, int64(dest.Value.Amount))
			if !ok {
				return errors.WithDetailf(errOverflow, "subtracting %d units of asset %x from mux destination %d from total %d underflows int64", dest.Value.Amount, dest.Value.AssetId.Bytes(), i, sum)
			}
			parity[*dest.Value.AssetId] = diff
		}

		for assetID, amount := range parity {
			if assetID == *consensus.BTMAssetID {
				if err = vs.gasStatus.setGas(amount, int64(vs.tx.SerializedSize)); err != nil {
					return err
				}
			} else if amount != 0 {
				return errors.WithDetailf(errUnbalanced, "asset %x sources - destinations = %d (should be 0)", assetID.Bytes(), amount)
			}
		}

		gasLeft, err := vm.Verify(NewTxVMContext(vs, e, e.Program, e.WitnessArguments), vs.gasStatus.GasLeft)
		if err != nil {
			return errors.Wrap(err, "checking mux program")
		}
		if err = vs.gasStatus.updateUsage(gasLeft); err != nil {
			return err
		}

		for _, BTMInputID := range vs.tx.GasInputIDs {
			e, ok := vs.tx.Entries[BTMInputID]
			if !ok {
				return errors.Wrapf(bc.ErrMissingEntry, "entry for bytom input %x not found", BTMInputID)
			}

			vs2 := *vs
			vs2.entryID = BTMInputID
			if err := checkValid(&vs2, e); err != nil {
				return errors.Wrap(err, "checking value source")
			}
		}

		for i, dest := range e.WitnessDestinations {
			vs2 := *vs
			vs2.destPos = uint64(i)
			err = checkValidDest(&vs2, dest)
			if err != nil {
				return errors.Wrapf(err, "checking mux destination %d", i)
			}
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

		if err := vs.gasStatus.setGasVaild(); err != nil {
			return err
		}

		for i, src := range e.Sources {
			vs2 := *vs
			vs2.sourcePos = uint64(i)
			err = checkValidSrc(&vs2, src)
			if err != nil {
				return errors.Wrapf(err, "checking mux source %d", i)
			}
		}

	case *bc.Nonce:
		//TODO: add block heigh range check on the control program
		gasLeft, err := vm.Verify(NewTxVMContext(vs, e, e.Program, e.WitnessArguments), vs.gasStatus.GasLeft)
		if err != nil {
			return errors.Wrap(err, "checking nonce program")
		}
		if err = vs.gasStatus.updateUsage(gasLeft); err != nil {
			return err
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Output:
		vs2 := *vs
		vs2.sourcePos = 0
		err = checkValidSrc(&vs2, e.Source)
		if err != nil {
			return errors.Wrap(err, "checking output source")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Retirement:
		vs2 := *vs
		vs2.sourcePos = 0
		err = checkValidSrc(&vs2, e.Source)
		if err != nil {
			return errors.Wrap(err, "checking retirement source")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Issuance:
		computedAssetID := e.WitnessAssetDefinition.ComputeAssetID()
		if computedAssetID != *e.Value.AssetId {
			return errors.WithDetailf(errMismatchedAssetID, "asset ID is %x, issuance wants %x", computedAssetID.Bytes(), e.Value.AssetId.Bytes())
		}

		anchor, ok := vs.tx.Entries[*e.AnchorId]
		if !ok {
			return errors.Wrapf(bc.ErrMissingEntry, "entry for issuance anchor %x not found", e.AnchorId.Bytes())
		}

		gasLeft, err := vm.Verify(NewTxVMContext(vs, e, e.WitnessAssetDefinition.IssuanceProgram, e.WitnessArguments), vs.gasStatus.GasLeft)
		if err != nil {
			return errors.Wrap(err, "checking issuance program")
		}
		if err = vs.gasStatus.updateUsage(gasLeft); err != nil {
			return err
		}

		var anchored *bc.Hash
		switch a := anchor.(type) {
		case *bc.Nonce:
			anchored = a.WitnessAnchoredId

		case *bc.Spend:
			anchored = a.WitnessAnchoredId

		case *bc.Issuance:
			anchored = a.WitnessAnchoredId

		default:
			return errors.WithDetailf(bc.ErrEntryType, "issuance anchor has type %T, should be nonce, spend, or issuance", anchor)
		}

		if *anchored != vs.entryID {
			return errors.WithDetailf(errMismatchedReference, "issuance %x anchor is for %x", vs.entryID.Bytes(), anchored.Bytes())
		}

		anchorVS := *vs
		anchorVS.entryID = *e.AnchorId
		err = checkValid(&anchorVS, anchor)
		if err != nil {
			return errors.Wrap(err, "checking issuance anchor")
		}

		destVS := *vs
		destVS.destPos = 0
		err = checkValidDest(&destVS, e.WitnessDestination)
		if err != nil {
			return errors.Wrap(err, "checking issuance destination")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Spend:
		if e.SpentOutputId == nil {
			return errors.Wrap(errMissingField, "spend without spent output ID")
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
				errMismatchedValue,
				"previous output is for %d unit(s) of %x, spend wants %d unit(s) of %x",
				spentOutput.Source.Value.Amount,
				spentOutput.Source.Value.AssetId.Bytes(),
				e.WitnessDestination.Value.Amount,
				e.WitnessDestination.Value.AssetId.Bytes(),
			)
		}

		vs2 := *vs
		vs2.destPos = 0
		err = checkValidDest(&vs2, e.WitnessDestination)
		if err != nil {
			return errors.Wrap(err, "checking spend destination")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	default:
		return fmt.Errorf("entry has unexpected type %T", e)
	}

	return nil
}

func checkValidSrc(vstate *validationState, vs *bc.ValueSource) error {
	if vs == nil {
		return errors.Wrap(errMissingField, "empty value source")
	}
	if vs.Ref == nil {
		return errors.Wrap(errMissingField, "missing ref on value source")
	}
	if vs.Value == nil || vs.Value.AssetId == nil {
		return errors.Wrap(errMissingField, "missing value on value source")
	}

	e, ok := vstate.tx.Entries[*vs.Ref]
	if !ok {
		return errors.Wrapf(bc.ErrMissingEntry, "entry for value source %x not found", vs.Ref.Bytes())
	}
	vstate2 := *vstate
	vstate2.entryID = *vs.Ref
	err := checkValid(&vstate2, e)
	if err != nil {
		return errors.Wrap(err, "checking value source")
	}

	var dest *bc.ValueDestination
	switch ref := e.(type) {
	case *bc.Coinbase:
		if vs.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for coinbase source", vs.Position)
		}
		dest = ref.WitnessDestination
	case *bc.Issuance:
		if vs.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for issuance source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Spend:
		if vs.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for spend source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Mux:
		if vs.Position >= uint64(len(ref.WitnessDestinations)) {
			return errors.Wrapf(errPosition, "invalid position %d for %d-destination mux source", vs.Position, len(ref.WitnessDestinations))
		}
		dest = ref.WitnessDestinations[vs.Position]

	default:
		return errors.Wrapf(bc.ErrEntryType, "value source is %T, should be coinbase, issuance, spend, or mux", e)
	}

	if dest.Ref == nil || *dest.Ref != vstate.entryID {
		return errors.Wrapf(errMismatchedReference, "value source for %x has disagreeing destination %x", vstate.entryID.Bytes(), dest.Ref.Bytes())
	}

	if dest.Position != vstate.sourcePos {
		return errors.Wrapf(errMismatchedPosition, "value source position %d disagrees with %d", dest.Position, vstate.sourcePos)
	}

	eq, err := dest.Value.Equal(vs.Value)
	if err != nil {
		return errors.Sub(errMissingField, err)
	}
	if !eq {
		return errors.Wrapf(errMismatchedValue, "source value %v disagrees with %v", dest.Value, vs.Value)
	}

	return nil
}

func checkValidDest(vs *validationState, vd *bc.ValueDestination) error {
	if vd == nil {
		return errors.Wrap(errMissingField, "empty value destination")
	}
	if vd.Ref == nil {
		return errors.Wrap(errMissingField, "missing ref on value destination")
	}
	if vd.Value == nil || vd.Value.AssetId == nil {
		return errors.Wrap(errMissingField, "missing value on value source")
	}

	e, ok := vs.tx.Entries[*vd.Ref]
	if !ok {
		return errors.Wrapf(bc.ErrMissingEntry, "entry for value destination %x not found", vd.Ref.Bytes())
	}
	var src *bc.ValueSource
	switch ref := e.(type) {
	case *bc.Output:
		if vd.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for output destination", vd.Position)
		}
		src = ref.Source

	case *bc.Retirement:
		if vd.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for retirement destination", vd.Position)
		}
		src = ref.Source

	case *bc.Mux:
		if vd.Position >= uint64(len(ref.Sources)) {
			return errors.Wrapf(errPosition, "invalid position %d for %d-source mux destination", vd.Position, len(ref.Sources))
		}
		src = ref.Sources[vd.Position]

	default:
		return errors.Wrapf(bc.ErrEntryType, "value destination is %T, should be output, retirement, or mux", e)
	}

	if src.Ref == nil || *src.Ref != vs.entryID {
		return errors.Wrapf(errMismatchedReference, "value destination for %x has disagreeing source %x", vs.entryID.Bytes(), src.Ref.Bytes())
	}

	if src.Position != vs.destPos {
		return errors.Wrapf(errMismatchedPosition, "value destination position %d disagrees with %d", src.Position, vs.destPos)
	}

	eq, err := src.Value.Equal(vd.Value)
	if err != nil {
		return errors.Sub(errMissingField, err)
	}
	if !eq {
		return errors.Wrapf(errMismatchedValue, "destination value %v disagrees with %v", src.Value, vd.Value)
	}

	return nil
}

// ValidateBlock validates a block and the transactions within.
// It does not run the consensus program; for that, see ValidateBlockSig.
func ValidateBlock(b, prev *bc.Block, seed *bc.Hash) error {
	if b.Height > 0 {
		if prev == nil {
			return errors.WithDetailf(errNoPrevBlock, "height %d", b.Height)
		}
		err := validateBlockAgainstPrev(b, prev)
		if err != nil {
			return err
		}
	}

	if b.Timestamp > uint64(time.Now().Unix())+consensus.MaxTimeOffsetSeconds {
		return errBadTimestamp
	}

	if !difficulty.CheckProofOfWork(&b.ID, seed, b.BlockHeader.Bits) {
		return errWorkProof
	}

	b.TransactionStatus = bc.NewTransactionStatus()
	coinbaseValue := consensus.BlockSubsidy(b.BlockHeader.Height)
	gasUsed := uint64(0)
	for i, tx := range b.Transactions {
		if b.Version == 1 && tx.Version != 1 {
			return errors.WithDetailf(errTxVersion, "block version %d, transaction version %d", b.Version, tx.Version)
		}
		if tx.TimeRange > b.Timestamp {
			return errors.New("invalid transaction time range")
		}
		gasStatus, err := ValidateTx(tx, b)
		gasOnlyTx := false
		if err != nil {
			if gasStatus == nil || !gasStatus.GasVaild {
				return errors.Wrapf(err, "validity of transaction %d of %d", i, len(b.Transactions))
			}
			gasOnlyTx = true
		}
		b.TransactionStatus.SetStatus(i, gasOnlyTx)
		coinbaseValue += gasStatus.BTMValue
		gasUsed += uint64(gasStatus.GasUsed)
	}

	if gasUsed > consensus.MaxBlockGas {
		return errOverBlockLimit
	}

	// check the coinbase output entry value
	if err := validateCoinbase(b.Transactions[0], coinbaseValue); err != nil {
		return err
	}

	txRoot, err := bc.MerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction merkle root")
	}

	if txRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "computed %x, current block wants %x", txRoot.Bytes(), b.TransactionsRoot.Bytes())
	}

	if bc.EntryID(b.TransactionStatus) != *b.TransactionStatusHash {
		return errMismatchedTxStatus
	}
	return nil
}

func validateCoinbase(tx *bc.Tx, value uint64) error {
	resultEntry := tx.Entries[*tx.TxHeader.ResultIds[0]]
	output, ok := resultEntry.(*bc.Output)
	if !ok {
		return errors.Wrap(errWrongCoinbaseTransaction, "decode output")
	}

	if output.Source.Value.Amount != value {
		return errors.Wrap(errWrongCoinbaseTransaction, "dismatch output value")
	}

	inputEntry := tx.Entries[tx.InputIDs[0]]
	input, ok := inputEntry.(*bc.Coinbase)
	if !ok {
		return errors.Wrap(errWrongCoinbaseTransaction, "decode input")
	}
	if input.Arbitrary != nil && len(input.Arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
		return errors.Wrap(errWrongCoinbaseTransaction, "coinbase arbitrary is over size")
	}
	return nil
}

func validateBlockAgainstPrev(b, prev *bc.Block) error {
	if b.Version < prev.Version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", prev.Version, b.Version)
	}
	if b.Height != prev.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", prev.Height, b.Height)
	}

	if prev.ID != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", prev.ID.Bytes(), b.PreviousBlockId.Bytes())
	}
	if b.Timestamp <= prev.Timestamp {
		return errors.WithDetailf(errMisorderedBlockTime, "previous block time %d, current block time %d", prev.Timestamp, b.Timestamp)
	}
	return nil
}

func validateStandardTx(tx *bc.Tx) error {
	for _, id := range tx.InputIDs {
		e, ok := tx.Entries[id]
		if !ok {
			return errors.New("miss tx input entry")
		}
		if spend, ok := e.(*bc.Spend); ok {
			if *spend.WitnessDestination.Value.AssetId != *consensus.BTMAssetID {
				continue
			}
			spentOutput, err := tx.Output(*spend.SpentOutputId)
			if err != nil {
				return errors.Wrap(err, "getting spend prevout")
			}

			if !segwit.IsP2WScript(spentOutput.ControlProgram.Code) {
				return errNotStandardTx
			}
		}
	}

	for _, id := range tx.ResultIds {
		e, ok := tx.Entries[*id]
		if !ok {
			return errors.New("miss tx output entry")
		}
		if output, ok := e.(*bc.Output); ok {
			if *output.Source.Value.AssetId != *consensus.BTMAssetID {
				continue
			}
			if !segwit.IsP2WScript(output.ControlProgram.Code) {
				return errNotStandardTx
			}
		}
	}
	return nil
}

// ValidateTx validates a transaction.
func ValidateTx(tx *bc.Tx, block *bc.Block) (*GasState, error) {
	if tx.TxHeader.SerializedSize > consensus.MaxTxSize || tx.TxHeader.SerializedSize == 0 {
		return nil, errWrongTransactionSize
	}
	if len(tx.ResultIds) == 0 {
		return nil, errors.New("tx didn't have any output")
	}

	if len(tx.GasInputIDs) == 0 && tx != block.Transactions[0] {
		return nil, errors.New("tx didn't have gas input")
	}

	if err := validateStandardTx(tx); err != nil {
		return nil, err
	}

	vs := &validationState{
		block:   block,
		tx:      tx,
		entryID: tx.ID,
		gasStatus: &GasState{
			GasVaild: false,
		},
		cache: make(map[bc.Hash]error),
	}

	err := checkValid(vs, tx.TxHeader)
	return vs.gasStatus, err
}
