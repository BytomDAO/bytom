package blockchain

import (
	"context"
	"testing"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pin"
	"github.com/bytom/blockchain/txdb"
	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
	dbm "github.com/tendermint/tmlibs/db"
)

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

	accountsDB := dbm.NewDB("account", config.DBBackend, config.DBDir())
	accUTXODB := dbm.NewDB("accountutxos", config.DBBackend, config.DBDir())
	pinStore = pin.NewStore(accUTXODB)

	err = pinStore.LoadAll(ctx)
	accounts = account.NewManager(accountsDB, chain, pinStore)

	assetsDB := dbm.NewDB("asset", config.DBBackend, config.DBDir())
	assets = asset.NewRegistry(assetsDB, chain)

	/*
	   c := prottest.NewChain(t)
	   assets := asset.NewRegistry(db, c, pinStore)
	   accounts å:= account.NewManager(db, c, pinStore)
	   coretest.CreatePins(ctx, t, pinStore)
	   accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	   go accounts.ProcessBlocks(ctx)
	   mockhsm := hsm.New(db)

	   xpub1, err := hsm.XCreate(ctx, "")
	   if err != nil {
	       t.Fatal(err)
	   }
	   acct1, err := accounts.Create(ctx, []chainkd.XPub{xpub1.XPub}, 1, "", nil, "")
	   if err != nil {
	       t.Fatal(err)
	   }

	   _, xpub2, err := chainkd.NewXKeys(nil)
	   if err != nil {
	       t.Fatal(err)
	   }
	   acct2, err := accounts.Create(ctx, []chainkd.XPub{xpub2}, 1, "", nil, "")
	   if err != nil {
	       t.Fatal(err)
	   }

	   assetDef1 := map[string]interface{}{"foo": 1}
	   assetDef2 := map[string]interface{}{"foo": 2}

	   asset1ID := coretest.CreateAsset(ctx, t, assets, assetDef1, "", nil)
	   asset2ID := coretest.CreateAsset(ctx, t, assets, assetDef2, "", nil)

	   issueSrc1 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetId: &asset1ID, Amount: 100}, nil))
	   issueSrc2 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetId: &asset2ID, Amount: 200}, nil))
	   issueDest1 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset1ID, Amount: 100}, acct1.ID, nil)
	   issueDest2 := accounts.NewControlAction(bc.AssetAmount{AssetId: &asset2ID, Amount: 200}, acct2.ID, nil)
	   tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{issueSrc1, issueSrc2, issueDest1, issueDest2}, time.Now().Add(time.Minute))
	   if err != nil {
	       t.Fatal(err)
	   }
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
