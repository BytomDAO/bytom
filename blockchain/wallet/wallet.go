package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

var walletKey = []byte("walletInfo")

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	Height uint64
	Hash   bc.Hash
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB     db.DB
	status StatusInfo
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB) *Wallet {
	w := &Wallet{
		DB: walletDB,
	}
	walletInfo, err := w.GetWalletInfo()
	if err != nil {
		log.WithField("warn", err).Warn("get wallet info")
	}
	w.status.Height = walletInfo.Height
	w.status.Hash = walletInfo.Hash
	return w
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) GetWalletInfo() (StatusInfo, error) {
	var info StatusInfo
	var rawWallet []byte

	if rawWallet = w.DB.Get(walletKey); rawWallet == nil {
		return info, nil
	}

	if err := json.Unmarshal(rawWallet, &info); err != nil {
		return info, err
	}

	return info, nil

}

func (w *Wallet) commitWalletInfo(batch *db.Batch) error {
	var info StatusInfo

	info.Height = w.status.Height
	info.Hash = w.status.Hash

	rawWallet, err := json.Marshal(info)
	if err != nil {
		log.WithField("err", err).Error("save wallet info")
		return err
	}
	//update wallet to db
	(*batch).Set(walletKey, rawWallet)
	//commit to db
	(*batch).Write()
	return nil
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) WalletUpdate(c *protocol.Chain) {
	var err error
	var block *legacy.Block

	storeBatch := w.DB.NewBatch()

LOOP:

	for !c.InMainChain(w.status.Height, w.status.Hash) {
		if block, err = c.GetBlockByHash(&w.status.Hash); err != nil {
			log.WithField("err", err).Error("get block by hash")
			return
		}

		//Reverse this block
		reverseAccountUTXOs(&storeBatch, block, w)
		deleteTransactions(&storeBatch, w.status.Height, block, w)
		log.WithField("Height", w.status.Height).Info("start rollback this block")

		w.status.Height = block.Height - 1
		w.status.Hash = block.PreviousBlockHash

		//update wallet info and commit batch write
		if err := w.commitWalletInfo(&storeBatch); err != nil {
			return
		}
	}

	block, _ = c.GetBlockByHeight(w.status.Height + 1)
	//if we already handled the tail of the chain, we wait
	if block == nil {
		<-c.BlockWaiter(w.status.Height + 1)
		if block, err = c.GetBlockByHeight(w.status.Height + 1); err != nil {
			log.WithField("err", err).Error("wallet get block by height")
			return
		}
	}

	//if false, means that rollback operation is necessary,then goto LOOP
	if block.PreviousBlockHash == w.status.Hash {
		//next loop will save
		w.status.Height = block.Height
		w.status.Hash = block.Hash()

		indexTransactions(&storeBatch, block, w)
		buildAccountUTXOs(&storeBatch, block, w)

		//update wallet info and commit batch write
		if err := w.commitWalletInfo(&storeBatch); err != nil {
			return
		}
	}

	//goto next loop
	goto LOOP
}
