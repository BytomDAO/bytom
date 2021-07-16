package validation

import (
	"bytes"

	"github.com/bytom/bytom/consensus/bcrp"
	"github.com/bytom/bytom/consensus/segwit"
	"github.com/bytom/bytom/crypto/sha3pool"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/vm"
)

// NewTxVMContext generates the vm.Context for BVM
func NewTxVMContext(vs *validationState, entry bc.Entry, prog *bc.Program, stateData [][]byte, args [][]byte) *vm.Context {
	var (
		tx          = vs.tx
		blockHeight = vs.block.BlockHeader.GetHeight()
		numResults  = uint64(len(tx.ResultIds))
		entryID     = bc.EntryID(entry) // TODO(bobg): pass this in, don't recompute it

		assetID       *[]byte
		amount        *uint64
		destPos       *uint64
		spentOutputID *[]byte
	)

	switch e := entry.(type) {
	case *bc.Issuance:
		a1 := e.Value.AssetId.Bytes()
		assetID = &a1
		amount = &e.Value.Amount
		destPos = &e.WitnessDestination.Position

	case *bc.Spend:
		spentOutput := tx.Entries[*e.SpentOutputId].(*bc.OriginalOutput)
		a1 := spentOutput.Source.Value.AssetId.Bytes()
		assetID = &a1
		amount = &spentOutput.Source.Value.Amount
		destPos = &e.WitnessDestination.Position
		s := e.SpentOutputId.Bytes()
		spentOutputID = &s
	}

	var txSigHash *[]byte
	txSigHashFn := func() []byte {
		if txSigHash == nil {
			hasher := sha3pool.Get256()
			defer sha3pool.Put256(hasher)

			entryID.WriteTo(hasher)
			tx.ID.WriteTo(hasher)

			var hash bc.Hash
			hash.ReadFrom(hasher)
			hashBytes := hash.Bytes()
			txSigHash = &hashBytes
		}
		return *txSigHash
	}

	ec := &entryContext{
		entry:   entry,
		entries: tx.Entries,
	}

	result := &vm.Context{
		VMVersion: prog.VmVersion,
		Code:      convertProgram(prog.Code, vs.converter),
		StateData: stateData,
		Arguments: args,

		EntryID: entryID.Bytes(),

		TxVersion:   &tx.Version,
		BlockHeight: &blockHeight,

		TxSigHash:     txSigHashFn,
		NumResults:    &numResults,
		AssetID:       assetID,
		Amount:        amount,
		DestPos:       destPos,
		SpentOutputID: spentOutputID,
		CheckOutput:   ec.checkOutput,
	}

	return result
}

func convertProgram(prog []byte, converter ProgramConverterFunc) []byte {
	if segwit.IsP2WPKHScript(prog) {
		if witnessProg, err := segwit.ConvertP2PKHSigProgram([]byte(prog)); err == nil {
			return witnessProg
		}
	} else if segwit.IsP2WSHScript(prog) {
		if witnessProg, err := segwit.ConvertP2SHProgram([]byte(prog)); err == nil {
			return witnessProg
		}
	} else if bcrp.IsCallContractScript(prog) {
		if contractProg, err := converter(prog); err == nil {
			return contractProg
		}
	}
	return prog
}

type entryContext struct {
	entry   bc.Entry
	entries map[bc.Hash]bc.Entry
}

func (ec *entryContext) checkOutput(index uint64, amount uint64, assetID []byte, vmVersion uint64, code []byte, state [][]byte, expansion bool) (bool, error) {
	checkEntry := func(e bc.Entry) (bool, error) {
		check := func(prog *bc.Program, value *bc.AssetAmount, stateData [][]byte) bool {
			return (prog.VmVersion == vmVersion &&
				bytes.Equal(prog.Code, code) &&
				bytes.Equal(value.AssetId.Bytes(), assetID) &&
				value.Amount == amount &&
				bytesEqual(stateData, state))
		}

		switch e := e.(type) {
		case *bc.OriginalOutput:
			return check(e.ControlProgram, e.Source.Value, e.StateData), nil

		case *bc.VoteOutput:
			return check(e.ControlProgram, e.Source.Value, e.StateData), nil

		case *bc.Retirement:
			var prog bc.Program
			if expansion {
				// The spec requires prog.Code to be the empty string only
				// when !expansion. When expansion is true, we prepopulate
				// prog.Code to give check() a freebie match.
				//
				// (The spec always requires prog.VmVersion to be zero.)
				prog.Code = code
			}
			return check(&prog, e.Source.Value, [][]byte{}), nil
		}

		return false, vm.ErrContext
	}

	checkMux := func(m *bc.Mux) (bool, error) {
		if index >= uint64(len(m.WitnessDestinations)) {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= %d", index, len(m.WitnessDestinations))
		}
		eID := m.WitnessDestinations[index].Ref
		e, ok := ec.entries[*eID]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for mux destination %d, id %x, not found", index, eID.Bytes())
		}
		return checkEntry(e)
	}

	switch e := ec.entry.(type) {
	case *bc.Mux:
		return checkMux(e)

	case *bc.Issuance:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for issuance destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)

	case *bc.Spend:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for spend destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)

	case *bc.VetoInput:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for vetoInput destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)

	}

	return false, vm.ErrContext
}

func bytesEqual(a, b [][]byte) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if !bytes.Equal(v, b[i]) {
			return false
		}
	}

	return true
}
