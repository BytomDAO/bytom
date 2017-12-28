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
	chain  *protocol.Chain
	status StatusInfo
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, chain *protocol.Chain) (*Wallet, error) {
	w := &Wallet{
		DB:    walletDB,
		chain: chain,
	}

	if err := w.loadWalletInfo(); err != nil {
		return nil, err
	}

	go w.walletUpdater()
	return w, nil
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) loadWalletInfo() error {
	if rawWallet := w.DB.Get(walletKey); rawWallet != nil {
		return json.Unmarshal(rawWallet, &w.status)
	}

	block, err := w.chain.GetBlockByHeight(0)
	if err != nil {
		return err
	}
	if err := w.attachBlock(block); err != nil {
		return err
	}
	return nil
}

func (w *Wallet) commitWalletInfo(batch db.Batch) error {
	rawWallet, err := json.Marshal(w.status)
	if err != nil {
		log.WithField("err", err).Error("save wallet info")
		return err
	}

	batch.Set(walletKey, rawWallet)
	batch.Write()
	return nil
}

func (w *Wallet) attachBlock(block *legacy.Block) error {
	if block.PreviousBlockHash != w.status.Hash {
		log.Warn("wallet skip attachBlock due to status hash not equal to previous hash")
	}

	storeBatch := w.DB.NewBatch()
	w.indexTransactions(storeBatch, block)
	w.buildAccountUTXOs(storeBatch, block)

	w.status.Height = block.Height
	w.status.Hash = block.Hash()
	return w.commitWalletInfo(storeBatch)
}

func (w *Wallet) detachBlock(block *legacy.Block) error {
	storeBatch := w.DB.NewBatch()
	w.reverseAccountUTXOs(storeBatch, block)
	w.deleteTransactions(storeBatch, w.status.Height, block)

	w.status.Height = block.Height - 1
	w.status.Hash = block.PreviousBlockHash
	return w.commitWalletInfo(storeBatch)
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) walletUpdater() {
	for {
		for !w.chain.InMainChain(w.status.Height, w.status.Hash) {
			block, err := w.chain.GetBlockByHash(&w.status.Hash)
			if err != nil {
				log.WithField("err", err).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.detachBlock(block); err != nil {
				log.WithField("err", err).Error("walletUpdater detachBlock")
				return
			}
		}

		block, _ := w.chain.GetBlockByHeight(w.status.Height + 1)
		if block == nil {
			<-w.chain.BlockWaiter(w.status.Height + 1)
			continue
		}

		if err := w.attachBlock(block); err != nil {
			log.WithField("err", err).Error("walletUpdater stop")
			return
		}
	}
}
