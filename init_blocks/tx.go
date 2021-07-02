package initblocks

import (
	"encoding/hex"
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/common"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

func NewAssetID(str string) bc.AssetID {
	assetBytes, err := hex.DecodeString(str)
	if err != nil {
		log.Fatal(err)
	}

	var bs [32]byte
	copy(bs[:], assetBytes)
	return bc.NewAssetID(bs)
}

func AddressToControlProgram(addressStr string) ([]byte, error) {
	address, err := common.DecodeAddress(addressStr, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}

	program := []byte{}
	redeemContract := address.ScriptAddress()
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return nil, errors.New("unsupport address type")
	}
	if err != nil {
		return nil, err
	}

	return program, nil
}

func getTxOriginalOutput(tx *types.Tx, i int) *bc.OriginalOutput {
	hash := tx.ResultIds[i]
	lastOutput, ok := tx.Entries[*hash]
	if !ok {
		log.Fatal("not exist of output hash " + hash.String())
	}

	originOutput, ok := lastOutput.(*bc.OriginalOutput)
	if !ok {
		log.Fatal("can not assert to bc.OriginalOutput pointer")
	}

	return originOutput
}

func buildOutputs(assetID bc.AssetID, addrBalances []AddressBalance) []*types.TxOutput {
	var outputs []*types.TxOutput
	for _, addrBalance := range addrBalances {
		controlProgram, err := AddressToControlProgram(addrBalance.Address)
		if err != nil {
			log.Fatal(err)
		}

		output := types.NewOriginalTxOutput(assetID, addrBalance.Balance, controlProgram, nil)
		outputs = append(outputs, output)
	}
	return outputs
}

func buildGenesisTx(assetTotals []AssetTotal) *types.Tx {
	contract, err := hex.DecodeString("00148c9d063ff74ee6d9ffa88d83aeb038068366c4c4")
	if err != nil {
		log.Panicf("fail on decode genesis tx output control program")
	}

	var outputs []*types.TxOutput
	for _, assetTotal := range assetTotals {
		output := types.NewOriginalTxOutput(NewAssetID(assetTotal.Asset), assetTotal.Amount, contract, nil)
		outputs = append(outputs, output)
	}

	txData := types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
		},
		Outputs: outputs,
	}
	return types.NewTx(txData)
}

func buildAllTxs(assetTotals []AssetTotal, asset2distributions map[string][]AddressBalance) []*types.Tx {
	var allTXs []*types.Tx
	genesisTx := buildGenesisTx(assetTotals)
	for i, output := range genesisTx.Outputs {
		addrBalances := asset2distributions[output.AssetId.String()]
		originOut := getTxOriginalOutput(genesisTx, i)
		assetTXs := buildAssetTXs(originOut, addrBalances)
		allTXs = append(allTXs, assetTXs...)
	}
	return allTXs
}

func sumBalance(addrBalances []AddressBalance) uint64 {
	sum := uint64(0)
	for _, addrBalance := range addrBalances {
		sum += addrBalance.Balance
	}
	return sum
}

func buildAssetTXs(output *bc.OriginalOutput, addrBalances []AddressBalance) []*types.Tx {
	preOut := output
	var txs []*types.Tx
	for i := 0; i < len(addrBalances); i += OutputCntPerTx {
		var batchAddrBalances []AddressBalance
		if len(addrBalances[i:]) < OutputCntPerTx {
			batchAddrBalances = addrBalances[i:]
		} else {
			batchAddrBalances = addrBalances[i : i+OutputCntPerTx]
		}

		outputs := buildOutputs(*preOut.Source.Value.AssetId, batchAddrBalances)
		leftBalance := preOut.Source.Value.Amount - sumBalance(addrBalances)
		if leftBalance < 0 {
			log.Fatal("left balance less zero")
		}

		changeOutput := types.NewOriginalTxOutput(*preOut.Source.Value.AssetId, leftBalance, preOut.ControlProgram.Code, nil)
		outputs = append(outputs, changeOutput)

		txData := types.TxData{
			Version: 1,
			Inputs: []*types.TxInput{
				types.NewSpendInput(
					nil,
					*preOut.Source.Ref,
					*preOut.Source.Value.AssetId,
					preOut.Source.Value.Amount,
					preOut.Source.Position,
					preOut.ControlProgram.Code,
					preOut.StateData),
			},

			Outputs: outputs,
		}

		tx := types.NewTx(txData)
		txs = append(txs, tx)
		preOut = getTxOriginalOutput(tx, len(tx.Outputs)-1)
	}
	return txs
}
