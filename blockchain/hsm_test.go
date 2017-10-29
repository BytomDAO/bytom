package blockchain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pin"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/blockchain/txdb"
	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	dbm "github.com/tendermint/tmlibs/db"
)

const dirPath = "pseudohsm/testdata/pseudo"

func TestHSM(t *testing.T) {
	ctx := context.Background()
	config := cfg.DefaultConfig()
	tc := dbm.NewDB("txdb", config.DBBackend, config.DBDir())
	store := txdb.NewStore(tc)

	var accounts *account.Manager
	var assets *asset.Registry
	var pinStore *pin.Store

	genesisBlock := &legacy.Block{
		BlockHeader:  legacy.BlockHeader{},
		Transactions: []*legacy.Tx{},
	}
	genesisBlock.UnmarshalText(consensus.InitBlock())
	txPool := protocol.NewTxPool()
	chain, err := protocol.NewChain(ctx, genesisBlock.Hash(), store, txPool, nil)
	if err != nil {
		t.Fatal(err)
	}
	accUTXODB := dbm.NewDB("accountutxos", config.DBBackend, config.DBDir())
	pinStore = pin.NewStore(accUTXODB)

	err = pinStore.LoadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	accountsDB := dbm.NewDB("account", config.DBBackend, config.DBDir())
	accounts = account.NewManager(accountsDB, chain, pinStore)

	assetsDB := dbm.NewDB("asset", config.DBBackend, config.DBDir())
	assets = asset.NewRegistry(assetsDB, chain)

	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}
	xpub1, err := hsm.XCreate("xpub1", "password")
	if err != nil {
		t.Fatal(err)
	}
	xpub2, err := hsm.XCreate("xpub2", "password")
	if err != nil {
		t.Fatal(err)
	}

	acct1, err := accounts.Create(ctx, []chainkd.XPub{xpub1.XPub}, 1, "acc1", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	acct2, err := accounts.Create(ctx, []chainkd.XPub{xpub2.XPub}, 1, "acc2", nil, "")
	if err != nil {
		t.Fatal(err)
	}

	assetDef1 := map[string]interface{}{"foo": 1}
	assetDef2 := map[string]interface{}{"foo": 2}

	asset1, err := assets.Define(ctx, []chainkd.XPub{xpub1.XPub}, 1, assetDef1, "foo1", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	asset2, err := assets.Define(ctx, []chainkd.XPub{xpub2.XPub}, 1, assetDef2, "foo2", nil, "")
	if err != nil {
		t.Fatal(err)
	}

	issue1 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetId: &asset1.AssetID, Amount: 100}, nil))
	issue2 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetId: &asset2.AssetID, Amount: 200}, nil))
	spend1 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset1.AssetID, Amount: 100}, acct1.ID, nil)
	spend2 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset2.AssetID, Amount: 200}, acct2.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{issue1, issue2, spend1, spend2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	err = txbuilder.Sign(ctx, tmpl, []chainkd.XPub{xpub1.XPub, xpub2.XPub}, "password", func(_ context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
		sigBytes, err := hsm.XSign(xpub, path, data[:], password)
		if err != nil {
			return nil, nil
		}
		return sigBytes, err
	})
	fmt.Printf("###data: %v#####", *tmpl)

	/*
	   c := prottest.NewChain(t)
	   assets := asset.NewRegistry(db, c, pinStore)
	   accounts å:= account.NewManager(db, c, pinStore)
	   coretest.CreatePins(ctx, t, pinStore)
	   accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	   go accounts.ProcessBlocks(ctx)

	   coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	   err = txbuilder.FinalizeTx(ctx, c, g, tmpl.Transaction)
	   if err != nil {
	       t.Fatal(err)
	   }

	   // Make a block so that UTXOs from the above tx are available to spend.
	   prottest.MakeBlock(t, c, g.PendingTxs())
	   <-pinStore.PinWaiter(account.PinName, c.Height())

	   xferSrc1 := accounts.NewSpendAction(bc.AssetAmount{AssetId: &asset1ID, Amount: 10}, acct1.ID, nil, nil)
	   xferSrc2 := accounts.NewSpendAction(bc.AssetAmount{AssetId: &asset2ID, Amount: 20}, acct2.ID, nil, nil)
	   xferDest1 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset2ID, Amount: 20}, acct1.ID, nil)
	   xferDest2 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset1ID, Amount: 10}, acct2.ID, nil)
	   tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{xferSrc1, xferSrc2, xferDest1, xferDest2}, time.Now().Add(time.Minute))
	   if err != nil {
	       t.Fatal(err)
	   }
	*/
}
