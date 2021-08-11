package config

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

// AssetIssue asset issue params
type AssetIssue struct {
	NonceStr           string
	IssuanceProgramStr string
	AssetDefinitionStr string
	AssetIDStr         string
	Amount             uint64
}

func (a *AssetIssue) nonce() []byte {
	bs, err := hex.DecodeString(a.NonceStr)
	if err != nil {
		panic(err)
	}

	return bs
}

func (a *AssetIssue) issuanceProgram() []byte {
	bs, err := hex.DecodeString(a.IssuanceProgramStr)
	if err != nil {
		panic(err)
	}

	return bs
}

func (a *AssetIssue) assetDefinition() []byte {
	bs, err := hex.DecodeString(a.AssetDefinitionStr)
	if err != nil {
		panic(err)
	}

	return bs
}

func (a *AssetIssue) id() bc.AssetID {
	var bs [32]byte
	bytes, err := hex.DecodeString(a.AssetIDStr)
	if err != nil {
		panic(err)
	}

	copy(bs[:], bytes)
	return bc.NewAssetID(bs)
}

// GenesisTxs make genesis block txs
func GenesisTxs() []*types.Tx {
	contract, err := hex.DecodeString("00148dea804c8f5a27518be2696e43dca0352b9a93cc")
	if err != nil {
		log.Panicf("fail on decode genesis tx output control program")
	}

	var txs []*types.Tx
	firstTxData := types.TxData{
		Version: 1,
		Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018."))},
		Outputs: []*types.TxOutput{types.NewOriginalTxOutput(*consensus.BTMAssetID, consensus.InitBTMSupply, contract, nil)},
	}
	txs = append(txs, types.NewTx(firstTxData))

	inputs := []*types.TxInput{}
	outputs := []*types.TxOutput{}
	for _, asset := range assetIssues {
		inputs = append(inputs, types.NewIssuanceInput(asset.nonce(), asset.Amount, asset.issuanceProgram(), nil, asset.assetDefinition()))
		outputs = append(outputs, types.NewOriginalTxOutput(asset.id(), asset.Amount, contract, nil))
	}

	secondTxData := types.TxData{Version: 1, Inputs: inputs, Outputs: outputs}
	txs = append(txs, types.NewTx(secondTxData))
	return txs
}

var assetIssues = []*AssetIssue{
	{
		NonceStr:           "8e972359c6441299",
		IssuanceProgramStr: "ae20d66ab117eca2bba6aefed569e52d6bf68a7a4ad7d775cbc01f7b2cfcd798f7b22031ab3c2147c330c5e360b4e585047d1dea5f529476ad5aff475ecdd55541923120851b4a24975df6dbeb4f8e5348542764f85bed67b763875325aa5e45116751b35253ad",
		AssetDefinitionStr: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a20224d4147222c0a20202271756f72756d223a20322c0a20202272656973737565223a202274727565222c0a20202273796d626f6c223a20224d4147220a7d",
		AssetIDStr:         "a0a71c215764e342d10d003be6369baf4145d9c7977f7b8f6bf446e628d8b8b8",
		Amount:             100000000000000,
	},
	{
		NonceStr:           "f57f15fbbdf5f70d",
		IssuanceProgramStr: "0354df07cda069ae203c939b7ba615a49adf57b5b4c9c37e80666436970f8507d56a272539cabcb9e15151ad",
		AssetDefinitionStr: "7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a207b7d2c0a2020226e616d65223a2022535550222c0a20202271756f72756d223a20312c0a20202272656973737565223a202266616c7365222c0a20202273796d626f6c223a2022535550220a7d",
		AssetIDStr:         "47fcd4d7c22d1d38931a6cd7767156babbd5f05bbbb3f7d3900635b56eb1b67e",
		Amount:             9866701291045,
	},
}
