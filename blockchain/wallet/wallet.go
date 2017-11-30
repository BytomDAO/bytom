package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

var walletkey = []byte("walletInfo")

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	Height uint64
	Hash   bc.Hash
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB       db.DB
	accounts *account.Manager
	assets   *asset.Registry
	Ind      *query.Indexer
	StatusInfo
}

//GlobalWallet for sourceReserve heightFn
var GlobalWallet Wallet

//InitWallet return a new wallet instance
func InitWallet(walletDB db.DB, accounts *account.Manager, assets *asset.Registry) *Wallet {
	GlobalWallet.DB = walletDB
	GlobalWallet.accounts = accounts
	GlobalWallet.assets = assets
	GlobalWallet.Ind = query.NewIndexer(walletDB)

	w := &GlobalWallet
	walletInfo, err := w.GetWalletInfo()
	if err != nil {
		log.WithField("warn", err).Warn("get wallet info")
	}
	w.Height = walletInfo.Height
	w.Hash = walletInfo.Hash
	return w
}

//GetWalletHeight return wallet on current height
func GetWalletHeight() uint64 {
	return GlobalWallet.Height
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) GetWalletInfo() (StatusInfo, error) {
	var info StatusInfo
	var rawWallet []byte

	if rawWallet = w.DB.Get(walletkey); rawWallet == nil {
		return info, nil
	}

	if err := json.Unmarshal(rawWallet, &w); err != nil {
		return info, err
	}

	return info, nil

}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) WalletUpdate(c *protocol.Chain) {
	var err error
	var block *legacy.Block

	storeBatch := w.DB.NewBatch()

LOOP:

	for !c.InMainChain(w.Height, w.Hash) {
		if block, err = c.GetBlockByHash(&w.Hash); err != nil {
			log.WithField("err", err).Error("get block by hash")
			return
		}

		//Reverse this block
		w.accounts.ReverseAccountUTXOs(&storeBatch, block)
		w.Ind.DeleteTransactions(w.Height, block)
		log.WithField("Height", w.Height).Info("start rollback this block")

		w.Height = block.Height - 1
		w.Hash = block.PreviousBlockHash

	}

	//update wallet info and commit batch write
	w.commitWalletInfo(&storeBatch)

	block, _ = c.GetBlockByHeight(w.Height + 1)
	//if we already handled the tail of the chain, we wait
	if block == nil {
		<-c.BlockWaiter(w.Height + 1)
		if block, err = c.GetBlockByHeight(w.Height + 1); err != nil {
			log.WithField("err", err).Error("wallet get block by height")
			return
		}
	}

	//if false, means that rollback operation is necessary,then goto LOOP
	if block.PreviousBlockHash == w.Hash {
		//next loop will save
		w.Height = block.Height
		w.Hash = block.Hash()

		w.assets.IndexAssets(block)
		w.accounts.BuildAccountUTXOs(&storeBatch, block)

		//update wallet info and commit batch write
		w.commitWalletInfo(&storeBatch)

		w.Ind.IndexTransactions(block)
	}

	//goto next loop
	goto LOOP
}

func (w *Wallet) commitWalletInfo(batch *db.Batch) {
	var info StatusInfo

	info.Height = w.Height
	info.Hash = w.Hash

	rawWallet, err := json.Marshal(info)
	if err != nil {
		log.WithField("err", err).Error("save wallet info")
		return
	}
	//update wallet to db
	(*batch).Set(walletkey, rawWallet)
	//commit to db
	(*batch).Write()
}
