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
func TestEditMargin(t *testing.T) {
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
		testutil.MustDecodeHexString("74202524d9fdf913c2b176f9c81c3f7d433440d944f12a87f9d78f3294b30e1d8a388716887ca20c012064059054e1c036e15d2da65441ff93bcb4593e374e09"),
		vm.Uint64Bytes(300000000),
		vm.Uint64Bytes(1),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("bcdda03bd6b1f5605c51e58cb1230c76fdc06b837118741a586275c6653979cd632f796741d79c652a06c23f73a0e2f7d9086ea39b4e9d9793e8835b5e013d07"),
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
		testutil.MustDecodeHexString("dda495953ff63af7775bfd8ad1b8b54900849a202668d35454beb6d33ae18057abbfc2a8f5691876083a364f713f54ea4b71cb9d0436d7b1c9ef194ee42e2304"),
		vm.Uint64Bytes(0),
	}

	arguments2 := [][]byte{
		testutil.MustDecodeHexString("2efd75e44777be73300210569bb4002e8942064718092a958658150f54f6a002806123a20960bbb5ce4d85cbff7827ca994344d360119041d266e3a56ddde904"),
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
