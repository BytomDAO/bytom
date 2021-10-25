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
	ownerScirpt  = testutil.MustDecodeHexString("0100000000000000000000000000000000000000000000000000000000000000") //[]byte("ownerScirpt")
	buyerScirpt  = testutil.MustDecodeHexString("0100000000000000000000000000000000000000000000000000000000000000") //[]byte("buyerScirpt")
)

// 从2个BTC的押金换成3个BTC的
func TestEditMargin(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	newStateData := [][]byte{
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	arguments := [][]byte{
		testutil.MustDecodeHexString("6c25b3220df660b06bf17ca881e6a31811fc4f33f78e5c0597a1e29b5d0030d9494bff417a237b907eac5cdce1dc0dfe1f258103dee32fcc84a2dd5e8c614209"),
		vm.Uint64Bytes(300000000),
		vm.Uint64Bytes(1),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments, utxoSourceID, nftAsset, 1, 0, contract, oldStateData),
			types.NewSpendInput(arguments, utxoSourceID, BTC, 200000000, 1, contract, oldStateData),
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

// 10个ETH质押被买走, 然后被质押15个ETH
func TestRegularBuy(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
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
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
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
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
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
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments := [][]byte{
		testutil.MustDecodeHexString("2dbbae17e2aed7639ba212f46fd28b9ccb122c1a24ebd1a48e2dddcaf675252e9adb7c8fe504fa4aad4d054d87701d93d67c5e62bee9ccdc442a39a207dcd106"),
		vm.Uint64Bytes(0),
	}

	tx := types.NewTx(types.TxData{
		Version:        1,
		SerializedSize: 10000,
		Inputs: []*types.TxInput{
			types.NewSpendInput(arguments, utxoSourceID, ETH, 135000000000, 2, offer, newStateData),
			types.NewSpendInput(nil, utxoSourceID, *consensus.BTMAssetID, 100000000, 1, anyCanSpendScript, nil),
		},
		Outputs: []*types.TxOutput{
			types.NewOriginalTxOutput(ETH, 135000000000, ownerScirpt, newStateData),
		},
	})

	_, err = validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{}}, func(prog []byte) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
}
