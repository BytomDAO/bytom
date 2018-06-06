package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	//SINGLE single sign
	SINGLE = 1

	maxTxChanSize = 10000 // txChanSize is the size of channel listening to Txpool newTxCh
)

var walletKey = []byte("walletInfo")

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	WorkHeight uint64
	WorkHash   bc.Hash
	BestHeight uint64
	BestHash   bc.Hash
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB         db.DB
	status     StatusInfo
	AccountMgr *account.Manager
	AssetReg   *asset.Registry
	Hsm        *pseudohsm.HSM
	chain      *protocol.Chain
	rescanCh   chan struct{}
	newTxCh    chan *types.Tx
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, account *account.Manager, asset *asset.Registry, hsm *pseudohsm.HSM, chain *protocol.Chain) (*Wallet, error) {
	w := &Wallet{
		DB:         walletDB,
		AccountMgr: account,
		AssetReg:   asset,
		chain:      chain,
		Hsm:        hsm,
		rescanCh:   make(chan struct{}, 1),
		newTxCh:    make(chan *types.Tx, maxTxChanSize),
	}

	if err := w.loadWalletInfo(); err != nil {
		return nil, err
	}

	go w.walletUpdater()
	go w.UnconfirmedTxCollector()

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
	return w.AttachBlock(block)
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

// AttachBlock attach a new block
func (w *Wallet) AttachBlock(block *types.Block) error {
	if block.PreviousBlockHash != w.status.WorkHash {
		log.Warn("wallet skip attachBlock due to status hash not equal to previous hash")
		return nil
	}

	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	storeBatch := w.DB.NewBatch()
	w.indexTransactions(storeBatch, block, txStatus)
	w.buildAccountUTXOs(storeBatch, block, txStatus)

	w.status.WorkHeight = block.Height
	w.status.WorkHash = block.Hash()
	if w.status.WorkHeight >= w.status.BestHeight {
		w.status.BestHeight = w.status.WorkHeight
		w.status.BestHash = w.status.WorkHash
	}
	return w.commitWalletInfo(storeBatch)
}

// DetachBlock detach a block and rollback state
func (w *Wallet) DetachBlock(block *types.Block) error {
	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	storeBatch := w.DB.NewBatch()
	w.reverseAccountUTXOs(storeBatch, block, txStatus)
	w.deleteTransactions(storeBatch, w.status.BestHeight)

	w.status.BestHeight = block.Height - 1
	w.status.BestHash = block.PreviousBlockHash

	if w.status.WorkHeight > w.status.BestHeight {
		w.status.WorkHeight = w.status.BestHeight
		w.status.WorkHash = w.status.BestHash
	}

	return w.commitWalletInfo(storeBatch)
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) walletUpdater() {
	for {
		w.getRescanNotification()
		for !w.chain.InMainChain(w.status.BestHash) {
			block, err := w.chain.GetBlockByHash(&w.status.BestHash)
			if err != nil {
				log.WithField("err", err).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.DetachBlock(block); err != nil {
				log.WithField("err", err).Error("walletUpdater detachBlock stop")
				return
			}
		}

		block, _ := w.chain.GetBlockByHeight(w.status.WorkHeight + 1)
		if block == nil {
			<-w.chain.BlockWaiter(w.status.WorkHeight + 1)
			continue
		}

		if err := w.AttachBlock(block); err != nil {
			log.WithField("err", err).Error("walletUpdater AttachBlock stop")
			return
		}
	}
}

//RescanBlocks provide a trigger to rescan blocks
func (w *Wallet) RescanBlocks() {
	select {
	case w.rescanCh <- struct{}{}:
	default:
		return
	}
}

func (w *Wallet) getRescanNotification() {
	select {
	case <-w.rescanCh:
		block, _ := w.chain.GetBlockByHeight(0)
		w.status.WorkHash = bc.Hash{}
		w.AttachBlock(block)
	default:
		return
	}
}

// GetNewTxCh return a unconfirmed transaction feed channel
func (w *Wallet) GetNewTxCh() chan *types.Tx {
	return w.newTxCh
}

func (w *Wallet) UnconfirmedTxCollector() {
	for {
		w.SaveUnconfirmedTx(<-w.newTxCh)
	}
}

func (w *Wallet) createProgram(account *account.Account, XPub *pseudohsm.XPub, index uint64) error {
	for i := uint64(0); i < index; i++ {
		if _, err := w.AccountMgr.CreateAddress(nil, account.ID, false); err != nil {
			return err
		}
	}
	return nil
}
