package wallet

import (
	"encoding/json"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/event"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	//SINGLE single sign
	SINGLE    = 1
	logModule = "wallet"
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
	DB              db.DB
	rw              sync.RWMutex
	status          StatusInfo
	AccountMgr      *account.Manager
	AssetReg        *asset.Registry
	Hsm             *pseudohsm.HSM
	chain           *protocol.Chain
	RecoveryMgr     *recoveryManager
	eventDispatcher *event.Dispatcher
	txMsgSub        *event.Subscription

	rescanCh chan struct{}
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, account *account.Manager, asset *asset.Registry, hsm *pseudohsm.HSM, chain *protocol.Chain, dispatcher *event.Dispatcher) (*Wallet, error) {
	w := &Wallet{
		DB:              walletDB,
		AccountMgr:      account,
		AssetReg:        asset,
		chain:           chain,
		Hsm:             hsm,
		RecoveryMgr:     newRecoveryManager(walletDB, account),
		eventDispatcher: dispatcher,
		rescanCh:        make(chan struct{}, 1),
	}

	if err := w.loadWalletInfo(); err != nil {
		return nil, err
	}

	if err := w.RecoveryMgr.LoadStatusInfo(); err != nil {
		return nil, err
	}

	var err error
	w.txMsgSub, err = w.eventDispatcher.Subscribe(protocol.TxMsgEvent{})
	if err != nil {
		return nil, err
	}

	go w.walletUpdater()
	go w.delUnconfirmedTx()
	go w.memPoolTxQueryLoop()
	return w, nil
}

// memPoolTxQueryLoop constantly pass a transaction accepted by mempool to the wallet.
func (w *Wallet) memPoolTxQueryLoop() {
	for {
		select {
		case obj, ok := <-w.txMsgSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("tx pool tx msg subscription channel closed")
				return
			}

			ev, ok := obj.Data.(protocol.TxMsgEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}

			switch ev.TxMsg.MsgType {
			case protocol.MsgNewTx:
				w.AddUnconfirmedTx(ev.TxMsg.TxDesc)
			case protocol.MsgRemoveTx:
				w.RemoveUnconfirmedTx(ev.TxMsg.TxDesc)
			default:
				log.WithFields(log.Fields{"module": logModule}).Warn("got unknow message type from the txPool channel")
			}
		}
	}
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) loadWalletInfo() error {
	if rawWallet := w.DB.Get(walletKey); rawWallet != nil {
		if err := json.Unmarshal(rawWallet, &w.status); err != nil {
			return err
		}

		//handle the case than use replace the coreDB during status in fork chain
		if w.chain.BlockExist(&w.status.BestHash) {
			return nil
		}

		log.WithFields(log.Fields{"module": logModule}).Warn("reset the wallet status due to core doesn't have wallet best block")
		w.deleteAccountTxs()
		w.deleteUtxos()
		w.status = StatusInfo{}
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
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("save wallet info")
		return err
	}

	batch.Set(walletKey, rawWallet)
	batch.Write()
	return nil
}

// AttachBlock attach a new block
func (w *Wallet) AttachBlock(block *types.Block) error {
	w.rw.Lock()
	defer w.rw.Unlock()

	if block.PreviousBlockHash != w.status.WorkHash {
		log.Warn("wallet skip attachBlock due to status hash not equal to previous hash")
		return nil
	}

	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	if err := w.RecoveryMgr.FilterRecoveryTxs(block); err != nil {
		return err
	}

	storeBatch := w.DB.NewBatch()
	if err := w.indexTransactions(storeBatch, block, txStatus); err != nil {
		return err
	}

	w.attachUtxos(storeBatch, block, txStatus)
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
	w.rw.Lock()
	defer w.rw.Unlock()

	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	storeBatch := w.DB.NewBatch()
	w.detachUtxos(storeBatch, block, txStatus)
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
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.DetachBlock(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater detachBlock stop")
				return
			}
		}

		block, _ := w.chain.GetBlockByHeight(w.status.WorkHeight + 1)
		if block == nil {
			w.walletBlockWaiter()
			continue
		}

		if err := w.AttachBlock(block); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater AttachBlock stop")
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

// deleteAccountTxs deletes all txs in wallet
func (w *Wallet) deleteAccountTxs() {
	storeBatch := w.DB.NewBatch()

	txIter := w.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		storeBatch.Delete(txIter.Key())
	}

	txIndexIter := w.DB.IteratorPrefix([]byte(TxIndexPrefix))
	defer txIndexIter.Release()

	for txIndexIter.Next() {
		storeBatch.Delete(txIndexIter.Key())
	}

	storeBatch.Write()
}

func (w *Wallet) deleteUtxos() {
	storeBatch := w.DB.NewBatch()
	ruIter := w.DB.IteratorPrefix([]byte(account.UTXOPreFix))
	defer ruIter.Release()
	for ruIter.Next() {
		storeBatch.Delete(ruIter.Key())
	}

	suIter := w.DB.IteratorPrefix([]byte(account.SUTXOPrefix))
	defer suIter.Release()
	for suIter.Next() {
		storeBatch.Delete(suIter.Key())
	}
	storeBatch.Write()
}

// DeleteAccount deletes account matching accountID, then rescan wallet
func (w *Wallet) DeleteAccount(accountID string) (err error) {
	w.rw.Lock()
	defer w.rw.Unlock()

	if err := w.AccountMgr.DeleteAccount(accountID); err != nil {
		return err
	}

	w.deleteAccountTxs()
	w.RescanBlocks()
	return nil
}

func (w *Wallet) UpdateAccountAlias(accountID string, newAlias string) (err error) {
	w.rw.Lock()
	defer w.rw.Unlock()

	if err := w.AccountMgr.UpdateAccountAlias(accountID, newAlias); err != nil {
		return err
	}

	w.deleteAccountTxs()
	w.RescanBlocks()
	return nil
}

func (w *Wallet) getRescanNotification() {
	select {
	case <-w.rescanCh:
		w.setRescanStatus()
	default:
		return
	}
}

func (w *Wallet) setRescanStatus() {
	block, _ := w.chain.GetBlockByHeight(0)
	w.status.WorkHash = bc.Hash{}
	w.AttachBlock(block)
}

func (w *Wallet) walletBlockWaiter() {
	select {
	case <-w.chain.BlockWaiter(w.status.WorkHeight + 1):
	case <-w.rescanCh:
		w.setRescanStatus()
	}
}

// GetWalletStatusInfo return current wallet StatusInfo
func (w *Wallet) GetWalletStatusInfo() StatusInfo {
	w.rw.RLock()
	defer w.rw.RUnlock()

	return w.status
}
