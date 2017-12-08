package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	//TxPreFix is transactions prefix
	TxPreFix = "TXS:"
)

var walletKey = []byte("walletInfo")

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	Height uint64
	Hash   bc.Hash
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB       db.DB
	annotators []Annotator
	StatusInfo
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB) *Wallet {
	w := &Wallet{
		DB:       walletDB,
	}
	walletInfo, err := w.GetWalletInfo()
	if err != nil {
		log.WithField("warn", err).Warn("get wallet info")
	}
	w.Height = walletInfo.Height
	w.Hash = walletInfo.Hash
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
		w.ReverseAccountUTXOs(&storeBatch, block)
		w.deleteTransactions(w.Height, block)
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

		w.indexTransactions(block)
		w.BuildAccountUTXOs(&storeBatch, block)
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
	(*batch).Set(walletKey, rawWallet)
	//commit to db
	(*batch).Write()
}

// Annotator describes a function capable of adding annotations
// to transactions, inputs and outputs.
type Annotator func(txs []*query.AnnotatedTx,walletDB db.DB) error

// RegisterAnnotator adds an additional annotator capable of mutating
// the annotated transaction object.
func (w *Wallet) RegisterAnnotator(annotator Annotator) {
	w.annotators = append(w.annotators, annotator)
}

// indexTransactions is registered as a block callback on the Chain. It
// saves all annotated transactions to the database.
func (w *Wallet) indexTransactions(b *legacy.Block) {
	if err := w.insertAnnotatedTxs(b); err != nil {
		log.WithField("err", err).Error("indexing transactions")
	}
}

func calcAnnotatedKey(blockHeight uint64, position uint32) []byte {
	return []byte(fmt.Sprintf("%s%016x%08x", TxPreFix, blockHeight, position))
}

func calcDeletePreFix(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPreFix, blockHeight))
}

//deleteTransaction delete transactions when orphan block rollback
func (w *Wallet) deleteTransactions(height uint64, b *legacy.Block) {
	txIter := w.DB.IteratorPrefix(calcDeletePreFix(height))
	storeBatch := w.DB.NewBatch()
	defer txIter.Release()

	for txIter.Next() {
		storeBatch.Delete(txIter.Key())
	}
	//commit
	storeBatch.Write()
}

func (w *Wallet) insertAnnotatedTxs(b *legacy.Block) error {
	annotatedTxs := make([]*query.AnnotatedTx, 0, len(b.Transactions))

	// Build the fully annotated transactions.
	for pos, tx := range b.Transactions {
		annotatedTxs = append(annotatedTxs, query.BuildAnnotatedTransaction(tx, b, uint32(pos)))
	}

	for _, annotator := range w.annotators {
		if err := annotator(annotatedTxs,w.DB); err != nil {
			return errors.Wrap(err, "adding external annotations")
		}
	}

	storeBatch := w.DB.NewBatch()
	for pos, tx := range annotatedTxs {
		rawTx, err := json.MarshalIndent(tx, "", "    ")
		if err != nil {
			return errors.Wrap(err, "inserting annotated_txs to db")
		}

		storeBatch.Set(calcAnnotatedKey(b.Height, uint32(pos)), rawTx)
	}

	//commit the annotated txs to the database.
	storeBatch.Write()

	return nil
}
