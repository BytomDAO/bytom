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
		testutil.MustDecodeHexString("d51a48a7b5918de3cbf97de1a047af34fcb144c0308558a6ed5c4a12f4f338b9ccef23cd51937116412059569ba966cdc3c19fd58caa8aca4c2b9323e15cc90d"),
		vm.Uint64Bytes(300000000),
		vm.Uint64Bytes(1),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("62fb14d988c57692a8860c25915e2fc7dbf4143388ede4200e79cc6dbf8751274cc0192d914137b9bb9f36b89d5e5f5909d041699f3d5fc1b27348e106a1520e"),
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
		testutil.MustDecodeHexString("ba9772c480c349dc237f15e9606963224d7d1211072a79b4c34a045d19bb95a2bc20fba10f70097775a5b4e2a5c9f016ca96713fa19bdc5dfc0b4d21a9c3d407"),
		vm.Uint64Bytes(200000000),
		vm.Uint64Bytes(1),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("d7d82e40d5a6022aa2b7eb602ded566fa748e9549bd73146d7c2251401805f113ed2f5d76a5208ebfae989e5c7da37e976d59564bf05f9682b5057e163121407"),
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
		testutil.MustDecodeHexString("4afb322c65d4d966b20de6a9fb891a926bb04fb9dcdcced8bf58b0caffbc0ed7798d6874f25bc34cec2ffe5a017fa10701d8c81812d174a10bd3e7a1fc5cb300"),
		vm.Uint64Bytes(0),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("4f7b9b4f15dfba12e80789619d33491a7537a68667f75b6889d40aa7138c9c497d3aaff3f7a7830cf155562ecd2d500bbcd453d323bcb0926d0e91ea09f42507"),
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
