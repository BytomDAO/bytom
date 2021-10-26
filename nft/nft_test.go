package nft

import (
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/validation"
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/testutil"
)

var (
	marginFold        = uint64(10)
	taxRate           = uint64(1000)
	anyCanSpendScript = testutil.MustDecodeHexString("51")
	platformScript    = []byte("platformScript")
	createrScript     = []byte("createrScript")
	nftAsset          = testutil.MustDecodeAsset("a0a71c215764e342d10d003be6369baf4145d9c7977f7b8f6bf446e628d8b8b8")
	BTC               = testutil.MustDecodeAsset("bda946b3110fa46fd94346ce3f05f0760f1b9de72e238835bc4d19f9d64f1742")
	ETH               = testutil.MustDecodeAsset("78de44ffa1bce37b757c9eae8925b5f199dc4621b412ef0f3f46168865284a93")

	utxoSourceID = testutil.MustDecodeHash("762ec536ea64f71feac5fd4000a4807fc8e9d08d757889bd0206a02b79f9db8e")
	ownerScirpt  = []byte("ownerScirpt")
	buyerScirpt  = []byte("buyerScirpt")
	publicKey    = testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e")
)

// 从2个BTC的押金换成3个BTC的
func TestAddMargin(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	newStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("a56a34053c2d0aeea5d590df4aa7f0694da5faf7f5e4d8848087de6de1d273fcefd5c04ed9ec4d95e40747bc8d145c12d01a6b09a3284c27b66d714d0a3fc203"),
		vm.Uint64Bytes(300000000),
		vm.Uint64Bytes(1),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("5a4c7a9dd7eec3e230b4f09f44f786d5864374c8208cb88cf00139c6f3bd65596c573bd486f98f7934dac24a8c35da028cc90a6a8df4a5c982f0e9a225fbc603"),
		vm.Uint64Bytes(300000000),
		vm.Uint64Bytes(1),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments1, utxoSourceID, nftAsset, 1, 0, contract, oldStateData),
			types.NewSpendInput(arguments2, utxoSourceID, BTC, 200000000, 1, contract, oldStateData),
			types.NewSpendInput(nil, utxoSourceID, BTC, 100000000, 2, anyCanSpendScript, nil),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(nftAsset, 1, contract, newStateData),
			types.NewOriginalTxOutput(BTC, 300000000, contract, newStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestSubMargin(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	newStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("02b8224c5aa208101fee786a13a1db435255107eca68ff9ca436f668fc85e984351be3bef911fab3fa30f65db1f7c87c98a8bfa1392709dc5fae65316e4f4e05"),
		vm.Uint64Bytes(200000000),
		vm.Uint64Bytes(1),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("475a68712f1d5eac547221921f7e6ff4d2d1414ef0c9a0483f6eb0dac604fd330f34029886a3fba1f5e9dabcedc5ee0c062187a365cd9b81099f462a1aa02c06"),
		vm.Uint64Bytes(200000000),
		vm.Uint64Bytes(1),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments1, utxoSourceID, nftAsset, 1, 0, contract, oldStateData),
			types.NewSpendInput(arguments2, utxoSourceID, BTC, 300000000, 1, contract, oldStateData),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(nftAsset, 1, contract, newStateData),
			types.NewOriginalTxOutput(BTC, 200000000, contract, newStateData),
			types.NewOriginalTxOutput(BTC, 100000000, ownerScirpt, oldStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}

// 10个ETH质押被买走, 然后被质押15个ETH
func TestRegularBuy(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments := [][]byte{
		ETH.Bytes(),
		buyerScirpt,
		vm.Uint64Bytes(15000000000),
		vm.Uint64Bytes(0),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments, utxoSourceID, nftAsset, 1, 0, contract, oldStateData),
			types.NewSpendInput(arguments, utxoSourceID, ETH, 10000000000, 1, contract, oldStateData),
			types.NewSpendInput(nil, utxoSourceID, ETH, 115000000000, 2, anyCanSpendScript, nil),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(nftAsset, 1, contract, newStateData),
			types.NewOriginalTxOutput(ETH, 15000000000, contract, newStateData),
			types.NewOriginalTxOutput(ETH, 10000000000, createrScript, oldStateData),
			types.NewOriginalTxOutput(ETH, 1000000000, platformScript, oldStateData),
			types.NewOriginalTxOutput(ETH, 99000000000, ownerScirpt, oldStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}

// 10个ETH质押被买走, 然后被质押2个BTC
func TestBuySwapMargin(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	arguments := [][]byte{
		BTC.Bytes(),
		buyerScirpt,
		vm.Uint64Bytes(200000000),
		vm.Uint64Bytes(0),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments, utxoSourceID, nftAsset, 1, 0, contract, oldStateData),
			types.NewSpendInput(arguments, utxoSourceID, ETH, 10000000000, 1, contract, oldStateData),
			types.NewSpendInput(nil, utxoSourceID, ETH, 100000000000, 2, anyCanSpendScript, nil),
			types.NewSpendInput(nil, utxoSourceID, BTC, 200000000, 3, anyCanSpendScript, nil),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(nftAsset, 1, contract, newStateData),
			types.NewOriginalTxOutput(BTC, 200000000, contract, newStateData),
			types.NewOriginalTxOutput(ETH, 10000000000, createrScript, oldStateData),
			types.NewOriginalTxOutput(ETH, 1000000000, platformScript, oldStateData),
			types.NewOriginalTxOutput(ETH, 99000000000, ownerScirpt, oldStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}

// 10个ETH质押被买走, 然后被质押15个ETH
func TestOfferBuy(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	offer, err := NewOffer(contract)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments := [][]byte{
		ETH.Bytes(),
		buyerScirpt,
		vm.Uint64Bytes(15000000000),
		vm.Uint64Bytes(0),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments, utxoSourceID, nftAsset, 1, 0, contract, oldStateData),
			types.NewSpendInput(arguments, utxoSourceID, ETH, 10000000000, 1, contract, oldStateData),
			types.NewSpendInput([][]byte{vm.Uint64Bytes(1)}, utxoSourceID, ETH, 115000000000, 2, offer, newStateData),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(nftAsset, 1, contract, newStateData),
			types.NewOriginalTxOutput(ETH, 15000000000, contract, newStateData),
			types.NewOriginalTxOutput(ETH, 10000000000, createrScript, oldStateData),
			types.NewOriginalTxOutput(ETH, 1000000000, platformScript, oldStateData),
			types.NewOriginalTxOutput(ETH, 99000000000, ownerScirpt, oldStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelOffer(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	offer, err := NewOffer(contract)
	if err != nil {
		t.Fatal(err)
	}

	newStateData := [][]byte{
		publicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("471cf24e018b549d5740cc2dca0d34ed33f5706c3d660ec8bbc21393731d0da51814b305009f5300a0a863a0c4c7ca3d8cc5d38cc5e0ec11e7eeec5161496d02"),
		vm.Uint64Bytes(0),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("3318b9ea0e808db8925f60ffba42c68106263fa2189d7dde68ef1c6d0c0d0d5efd24aa7b0d3e960e81880dd35828341a17585c41ad4bd9de569c765aba39ca02"),
		vm.Uint64Bytes(0),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments1, utxoSourceID, ETH, 135000000000, 2, offer, newStateData),
			types.NewSpendInput(arguments2, utxoSourceID, BTC, 135000000000, 1, offer, newStateData),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(ETH, 135000000000, ownerScirpt, newStateData),
			types.NewOriginalTxOutput(BTC, 135000000000, ownerScirpt, newStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}
