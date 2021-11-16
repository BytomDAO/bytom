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

	utxoSourceID   = testutil.MustDecodeHash("762ec536ea64f71feac5fd4000a4807fc8e9d08d757889bd0206a02b79f9db8e")
	ownerScirpt    = []byte("ownerScirpt")
	buyerScirpt    = []byte("buyerScirpt")
	ownerPublicKey = testutil.MustDecodeHexString("7642ba797fd89d1f98a8559b4ca74123697dd4dee882955acd0da9010a80d64e")
	buyerPublicKey = testutil.MustDecodeHexString("a0a5b2a1148cc8eb4fbdb739574164ecd6be15f4da31b54b846de5c5c83815b3")
)

// 从2个BTC的押金换成3个BTC的
func TestAddMargin(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	newStateData := [][]byte{
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("66d3b6d1422626213c39c2045bc8f5505333796c7c976e2152ca25f691248750fffc84b702d45fa17f8d74cc1beab2d782a664685dfcc95a291eaa1d5d05ad00"),
		vm.Uint64Bytes(300000000),
		vm.Uint64Bytes(1),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("89ad84d8fdf7dbe5a21aaf1048ff9cb342bdb97c2c79d1428fe45e9fc1c31d980d0b9c642790931d6998d79d5a278439e4621795dd8a85627cf6a4fff363e506"),
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
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	newStateData := [][]byte{
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("9b6d86c36c3e86a576e3d1512074e2a45c0768541b57afc4b36e1f54d924b5aeba4b5176dde01f9a7360a6d06bd9711c8be58a2b7ec5005217522438bff9520a"),
		vm.Uint64Bytes(200000000),
		vm.Uint64Bytes(2),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("ae5e2d806e49a418e7b952b132dcb95d26076302a44812ecd6076af26ac1377cc20bf66cdf6d560e944ffc62b27d3b589bff306bfce33e217f0f1d84ae07cd06"),
		vm.Uint64Bytes(200000000),
		vm.Uint64Bytes(2),
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

func TestTransferNft(t *testing.T) {
	contract, err := NewContract(platformScript, marginFold)
	if err != nil {
		t.Fatal(err)
	}

	oldStateData := [][]byte{
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	newStateData := [][]byte{
		buyerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(300000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("060e8b5e0090709f02d3c4ba7f83e8f38c0661154f3a5c58d267d315fc734ba4e6bccaa2aef2d10fbaa980808fd4dc89faaef9d64627e061518f613766737d09"),
		buyerPublicKey,
		buyerScirpt,
		vm.Uint64Bytes(3),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("e05d5dc410721c307a7287c2705482a065b93fdb4f0925389db25fe132f216b2eb504ceb844dd462bdc3468ab233972a1ae33394fcf66b374fcc4935281b5b06"),
		buyerPublicKey,
		buyerScirpt,
		vm.Uint64Bytes(3),
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
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
		buyerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments := [][]byte{
		buyerPublicKey,
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
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
		buyerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		BTC.Bytes(),
		vm.Uint64Bytes(200000000),
	}

	arguments := [][]byte{
		buyerPublicKey,
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
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		ownerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(10000000000),
	}

	newStateData := [][]byte{
		buyerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments := [][]byte{
		buyerPublicKey,
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
		ownerPublicKey,
		createrScript,
		vm.Uint64Bytes(taxRate),
		nftAsset.Bytes(),
		buyerScirpt,
		ETH.Bytes(),
		vm.Uint64Bytes(15000000000),
	}

	arguments1 := [][]byte{
		testutil.MustDecodeHexString("56b45f220874f271d7b130372ee5d5f3c86a3dd253a3a5fc0dfe3497591589604760449defe9f6bc48bac09dcedee22c948f7adee65f37715edb4301f3e3760c"),
		vm.Uint64Bytes(0),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("1f591206a9a427e3ffe0c9f03cc971b60bc43511a0fff3a5d200a14fa9ef13c31c0219cc13e78f8b0b72adfdc1da3fc1e08483273f2de7f1210bb2f3f6d7dc00"),
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
