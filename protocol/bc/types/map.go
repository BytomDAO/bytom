package types

import (
	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
)

// MapTx converts a types TxData object into its entries-based
// representation.
func MapTx(oldTx *TxData) *bc.Tx {
	txID, txHeader, entries := mapTx(oldTx)
	tx := &bc.Tx{
		TxHeader: txHeader,
		ID:       txID,
		Entries:  entries,
		InputIDs: make([]bc.Hash, len(oldTx.Inputs)),
	}

	spentOutputIDs := make(map[bc.Hash]bool)
	for id, e := range entries {
		var ord uint64
		switch e := e.(type) {
		case *bc.Issuance:
			ord = e.Ordinal

		case *bc.Spend:
			ord = e.Ordinal
			spentOutputIDs[*e.SpentOutputId] = true
			if *e.WitnessDestination.Value.AssetId == *consensus.BTMAssetID {
				tx.GasInputIDs = append(tx.GasInputIDs, id)
			}

		case *bc.Coinbase:
			ord = 0

		default:
			continue
		}

		if ord >= uint64(len(oldTx.Inputs)) {
			continue
		}
		tx.InputIDs[ord] = id
	}

	for id := range spentOutputIDs {
		tx.SpentOutputIDs = append(tx.SpentOutputIDs, id)
	}
	return tx
}

func mapTx(tx *TxData) (headerID bc.Hash, hdr *bc.TxHeader, entryMap map[bc.Hash]bc.Entry) {
	entryMap = make(map[bc.Hash]bc.Entry)
	addEntry := func(e bc.Entry) *bc.Hash {
		id := bc.EntryID(e)
		entryMap[id] = e
		return &id
	}

	spends, issuances, coinbase, muxSources := mapInputs(tx, addEntry)
	muxID, mux := buildMux(muxSources, addEntry)
	connectInputsToMux(spends, issuances, coinbase, entryMap, muxID, mux)
	resultIDs := mapOutputs(tx, muxID, mux, addEntry)

	headerId, header := mapTxHeader(tx, resultIDs, addEntry)

	return *headerId, header, entryMap
}

func mapInputs(tx *TxData, addEntry func(e bc.Entry) *bc.Hash) (spends []*bc.Spend, issuances []*bc.Issuance, coinbase *bc.Coinbase, muxSources []*bc.ValueSource) {
	muxSources = make([]*bc.ValueSource, len(tx.Inputs))
	for i, input := range tx.Inputs {
		switch inp := input.TypedInput.(type) {
		case *IssuanceInput:
			issuance := mapIssuance(input, i)
			issuanceID := addEntry(issuance)
			issuances = append(issuances, issuance)
			muxSources[i] = buildValueSource(issuanceID, issuance.Value, 0)
		case *SpendInput:
			// create entry for prevOut
			prevOut := buildPrevOut(inp)
			prevOutID := addEntry(prevOut)
			// create entry for spend
			spend := mapSpend(prevOutID, i, inp)
			spendID := addEntry(spend)
			spends = append(spends, spend)
			muxSources[i] = buildValueSource(spendID, &inp.AssetAmount, 0)
		case *CoinbaseInput:
			coinbase = bc.NewCoinbase(inp.Arbitrary)
			coinbaseID := addEntry(coinbase)
			muxSources[i] = buildValueSource(coinbaseID, &tx.Outputs[0].AssetAmount, 0)
		}
	}
	return
}

func buildValueSource(ref *bc.Hash, amount *bc.AssetAmount, position uint64) *bc.ValueSource {
	return &bc.ValueSource{Ref: ref, Value: amount, Position: position}
}

func mapIssuance(input *TxInput, ordinal int) *bc.Issuance {
	inp := input.TypedInput.(*IssuanceInput)
	nonceHash := inp.NonceHash()
	assetDefHash := inp.AssetDefinitionHash()
	value := input.AssetAmount()
	args := inp.Arguments
	assetDef := &bc.AssetDefinition{
		Data: &assetDefHash,
		IssuanceProgram: &bc.Program{
			VmVersion: inp.VMVersion,
			Code:      inp.IssuanceProgram,
		},
	}

	issuance := bc.NewIssuance(&nonceHash, &value, uint64(ordinal))
	issuance.WitnessArguments = args
	issuance.WitnessAssetDefinition = assetDef
	return issuance
}

func buildPrevOut(inp *SpendInput) *bc.Output {
	prog := &bc.Program{VmVersion: inp.VMVersion, Code: inp.ControlProgram}
	src := buildValueSource(&inp.SourceID, &inp.AssetAmount, inp.SourcePosition)
	// ordinal doesn't matter for prevOuts, only for result outputs
	prevOut := bc.NewOutput(src, prog, 0)
	return prevOut
}

func mapSpend(prevOutID *bc.Hash, i int, inp *SpendInput) *bc.Spend {
	spend := bc.NewSpend(prevOutID, uint64(i))
	spend.WitnessArguments = inp.Arguments
	return spend
}

func buildMux(muxSources []*bc.ValueSource, addEntry func(e bc.Entry) *bc.Hash) (muxID *bc.Hash, mux *bc.Mux) {
	mux = bc.NewMux(muxSources, &bc.Program{VmVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	muxID = addEntry(mux)
	return
}

func connectInputsToMux(spends []*bc.Spend, issuances []*bc.Issuance, coinbase *bc.Coinbase, entryMap map[bc.Hash]bc.Entry, muxID *bc.Hash, mux *bc.Mux) {
	for _, spend := range spends {
		spentOutput := entryMap[*spend.SpentOutputId].(*bc.Output)
		spend.SetDestination(muxID, spentOutput.Source.Value, spend.Ordinal)
	}
	for _, issuance := range issuances {
		issuance.SetDestination(muxID, issuance.Value, issuance.Ordinal)
	}
	if coinbase != nil {
		coinbase.SetDestination(muxID, mux.Sources[0].Value, 0)
	}
}

func mapOutputs(tx *TxData, muxID *bc.Hash, mux *bc.Mux, addEntry func(e bc.Entry) *bc.Hash) []*bc.Hash {
	var resultIDs []*bc.Hash
	for i, out := range tx.Outputs {
		src := buildValueSource(muxID, &out.AssetAmount, uint64(i))

		var resultID *bc.Hash
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			r := bc.NewRetirement(src, uint64(i))
			resultID = addEntry(r)
		} else {
			// non-retirement
			prog := &bc.Program{VmVersion: out.VMVersion, Code: out.ControlProgram}
			o := bc.NewOutput(src, prog, uint64(i))
			resultID = addEntry(o)
		}

		dest := &bc.ValueDestination{Value: src.Value, Ref: resultID, Position: 0}
		mux.WitnessDestinations = append(mux.WitnessDestinations, dest)

		resultIDs = append(resultIDs, resultID)
	}
	return resultIDs
}

func mapTxHeader(tx *TxData, resultIDs []*bc.Hash, addEntry func(e bc.Entry) *bc.Hash) (headerID *bc.Hash, header *bc.TxHeader) {
	header = bc.NewTxHeader(tx.Version, tx.SerializedSize, tx.TimeRange, resultIDs)
	headerID = addEntry(header)
	return
}

func mapBlockHeader(old *BlockHeader) (bc.Hash, *bc.BlockHeader) {
	bh := bc.NewBlockHeader(old.Version, old.Height, &old.PreviousBlockHash, old.Timestamp, &old.TransactionsMerkleRoot, &old.TransactionStatusHash, old.Nonce, old.Bits)
	return bc.EntryID(bh), bh
}

// MapBlock converts a types block to bc block
func MapBlock(old *Block) *bc.Block {
	if old == nil {
		return nil
	}

	b := new(bc.Block)
	b.ID, b.BlockHeader = mapBlockHeader(&old.BlockHeader)
	for _, oldTx := range old.Transactions {
		b.Transactions = append(b.Transactions, oldTx.Tx)
	}
	return b
}
