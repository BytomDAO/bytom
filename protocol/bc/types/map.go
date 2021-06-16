package types

import (
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

type mapHelper struct {
	txData   *TxData
	entryMap map[bc.Hash]bc.Entry

	// data using during the map process
	spends    []*bc.Spend
	issuances []*bc.Issuance
	vetos     []*bc.VetoInput
	coinbase  *bc.Coinbase

	muxSources []*bc.ValueSource
	mux        *bc.Mux

	inputIDs       []bc.Hash
	spentOutputIDs []bc.Hash
	resultIDs      []*bc.Hash
}

func newMapHelper(txData *TxData) *mapHelper {
	return &mapHelper{
		txData:         txData,
		entryMap:       make(map[bc.Hash]bc.Entry),
		spends:         []*bc.Spend{},
		issuances:      []*bc.Issuance{},
		vetos:          []*bc.VetoInput{},
		muxSources:     make([]*bc.ValueSource, len(txData.Inputs)),
		inputIDs:       make([]bc.Hash, len(txData.Inputs)),
		spentOutputIDs: []bc.Hash{},
		resultIDs:      []*bc.Hash{},
	}
}

func (mh *mapHelper) addEntry(e bc.Entry) bc.Hash {
	id := bc.EntryID(e)
	mh.entryMap[id] = e
	return id
}

func (mh *mapHelper) generateTx() *bc.Tx {
	header := bc.NewTxHeader(mh.txData.Version, mh.txData.SerializedSize, mh.txData.TimeRange, mh.resultIDs)
	return &bc.Tx{
		TxHeader:       header,
		ID:             mh.addEntry(header),
		Entries:        mh.entryMap,
		InputIDs:       mh.inputIDs,
		SpentOutputIDs: mh.spentOutputIDs,
	}
}

func (mh *mapHelper) mapCoinbaseInput(i int, input *CoinbaseInput) {
	mh.coinbase = bc.NewCoinbase(input.Arbitrary)
	mh.inputIDs[i] = mh.addEntry(mh.coinbase)
	var totalAmount uint64
	for _, output := range mh.txData.Outputs {
		totalAmount += output.Amount
	}
	mh.muxSources[i] = &bc.ValueSource{
		Ref:   &mh.inputIDs[i],
		Value: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: totalAmount},
	}
}

func (mh *mapHelper) mapIssuanceInput(i int, input *IssuanceInput) {
	nonceHash := input.NonceHash()
	assetDefHash := input.AssetDefinitionHash()
	assetID := input.AssetID()
	value := bc.AssetAmount{AssetId: &assetID, Amount: input.Amount}

	issuance := bc.NewIssuance(&nonceHash, &value, uint64(i))
	issuance.WitnessAssetDefinition = &bc.AssetDefinition{
		Data: &assetDefHash,
		IssuanceProgram: &bc.Program{
			VmVersion: input.VMVersion,
			Code:      input.IssuanceProgram,
		},
	}

	issuance.WitnessArguments = input.Arguments
	mh.issuances = append(mh.issuances, issuance)
	mh.inputIDs[i] = mh.addEntry(issuance)
	mh.muxSources[i] = &bc.ValueSource{
		Ref:   &mh.inputIDs[i],
		Value: &value,
	}
}

func (mh *mapHelper) mapSpendInput(i int, input *SpendInput) {
	// create entry for prevout
	prog := &bc.Program{VmVersion: input.VMVersion, Code: input.ControlProgram}
	src := &bc.ValueSource{
		Ref:      &input.SourceID,
		Value:    &input.AssetAmount,
		Position: input.SourcePosition,
	}

	prevout := bc.NewOriginalOutput(src, prog, input.StateData, 0) // ordinal doesn't matter for prevouts, only for result outputs
	prevoutID := mh.addEntry(prevout)

	// create entry for spend
	spend := bc.NewSpend(&prevoutID, uint64(i))
	spend.WitnessArguments = input.Arguments
	mh.spends = append(mh.spends, spend)
	mh.inputIDs[i] = mh.addEntry(spend)
	mh.spentOutputIDs = append(mh.spentOutputIDs, prevoutID)
	mh.muxSources[i] = &bc.ValueSource{
		Ref:   &mh.inputIDs[i],
		Value: &input.AssetAmount,
	}
}

func (mh *mapHelper) mapVetoInput(i int, input *VetoInput) {
	prog := &bc.Program{VmVersion: input.VMVersion, Code: input.ControlProgram}
	src := &bc.ValueSource{
		Ref:      &input.SourceID,
		Value:    &input.AssetAmount,
		Position: input.SourcePosition,
	}

	prevout := bc.NewVoteOutput(src, prog, input.StateData, 0, input.Vote) // ordinal doesn't matter for prevouts, only for result outputs
	prevoutID := mh.addEntry(prevout)
	// create entry for VetoInput
	vetoInput := bc.NewVetoInput(&prevoutID, uint64(i))
	vetoInput.WitnessArguments = input.Arguments
	mh.vetos = append(mh.vetos, vetoInput)
	mh.inputIDs[i] = mh.addEntry(vetoInput)
	mh.spentOutputIDs = append(mh.spentOutputIDs, prevoutID)
	mh.muxSources[i] = &bc.ValueSource{
		Ref:   &mh.inputIDs[i],
		Value: &input.AssetAmount,
	}
}

func (mh *mapHelper) mapInputs() {
	for i, input := range mh.txData.Inputs {
		switch typedInput := input.TypedInput.(type) {
		case *IssuanceInput:
			mh.mapIssuanceInput(i, typedInput)
		case *SpendInput:
			mh.mapSpendInput(i, typedInput)
		case *VetoInput:
			mh.mapVetoInput(i, typedInput)
		case *CoinbaseInput:
			mh.mapCoinbaseInput(i, typedInput)
		default:
			panic("fail on handle transaction input")
		}
	}
}

func (mh *mapHelper) initMux() {
	mh.mux = bc.NewMux(mh.muxSources, &bc.Program{VmVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	muxID := mh.addEntry(mh.mux)

	// connect the inputs to the mux
	for _, spend := range mh.spends {
		spentOutput := mh.entryMap[*spend.SpentOutputId].(*bc.OriginalOutput)
		spend.SetDestination(&muxID, spentOutput.Source.Value, spend.Ordinal)
	}

	for _, vetoInput := range mh.vetos {
		voteOutput := mh.entryMap[*vetoInput.SpentOutputId].(*bc.VoteOutput)
		vetoInput.SetDestination(&muxID, voteOutput.Source.Value, vetoInput.Ordinal)
	}

	for _, issuance := range mh.issuances {
		issuance.SetDestination(&muxID, issuance.Value, issuance.Ordinal)
	}

	if mh.coinbase != nil {
		mh.coinbase.SetDestination(&muxID, mh.mux.Sources[0].Value, 0)
	}
}

func (mh *mapHelper) mapOutputs() {
	muxID := bc.EntryID(mh.mux)
	for i, out := range mh.txData.Outputs {
		src := &bc.ValueSource{Ref: &muxID, Value: &out.AssetAmount, Position: uint64(i)}
		prog := &bc.Program{out.VMVersion, out.ControlProgram}

		var resultID bc.Hash
		switch {
		case vmutil.IsUnspendable(out.ControlProgram):
			r := bc.NewRetirement(src, uint64(i))
			resultID = mh.addEntry(r)

		case out.OutputType() == OriginalOutputType:
			o := bc.NewOriginalOutput(src, prog, out.StateData, uint64(i))
			resultID = mh.addEntry(o)

		case out.OutputType() == VoteOutputType:
			voteOut, _ := out.TypedOutput.(*VoteOutput)
			v := bc.NewVoteOutput(src, prog, out.StateData, uint64(i), voteOut.Vote)
			resultID = mh.addEntry(v)

		default:
			panic("fail on handle transaction output")
		}

		mh.resultIDs = append(mh.resultIDs, &resultID)
		mh.mux.WitnessDestinations = append(mh.mux.WitnessDestinations, &bc.ValueDestination{
			Value:    src.Value,
			Ref:      &resultID,
			Position: 0,
		})
	}
}

// MapTx converts a types TxData object into its entries-based
// representation.
func MapTx(txData *TxData) *bc.Tx {
	mh := newMapHelper(txData)
	mh.mapInputs()
	mh.initMux()
	mh.mapOutputs()
	return mh.generateTx()
}

func mapBlockHeader(old *BlockHeader) (bc.Hash, *bc.BlockHeader) {
	bh := bc.NewBlockHeader(old.Version, old.Height, &old.PreviousBlockHash, old.Timestamp, &old.TransactionsMerkleRoot)
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
