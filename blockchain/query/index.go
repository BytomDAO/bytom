package query

import (
	"bytom/crypto/sha3pool"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	//TxPreFix is transactions prefix
	TxPreFix          = "TXS:"
	accountCPPreFix   = "ACP:"
	accountUTXOPreFix = "ACU:"
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(walletDB dbm.DB) *Indexer {
	indexer := &Indexer{
		db: walletDB,
	}
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db         dbm.DB
	annotators []Annotator
}

// Annotator describes a function capable of adding annotations
// to transactions, inputs and outputs.
type Annotator func(txs []*AnnotatedTx) error

// RegisterAnnotator adds an additional annotator capable of mutating
// the annotated transaction object.
func (ind *Indexer) RegisterAnnotator(annotator Annotator) {
	ind.annotators = append(ind.annotators, annotator)
}

// IndexTransactions is registered as a block callback on the Chain. It
// saves all annotated transactions to the database.
func (ind *Indexer) IndexTransactions(b *legacy.Block, account dbm.DB, wallet dbm.DB) {
	if err := ind.insertAnnotatedTxs(b, account, wallet); err != nil {
		logrus.WithField("err", err).Error("indexing transactions")
	}
}

func calcAnnotatedKey(blockHeight uint64, position uint32) []byte {
	return []byte(fmt.Sprintf("%s%016x%08x", TxPreFix, blockHeight, position))
}

func calcDeletePreFix(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPreFix, blockHeight))
}

//DeleteTransaction delete transactions when orphan block rollback
func (ind *Indexer) DeleteTransactions(height uint64, b *legacy.Block) {
	txIter := ind.db.IteratorPrefix(calcDeletePreFix(height))
	storeBatch := ind.db.NewBatch()
	defer txIter.Release()

	for txIter.Next() {
		storeBatch.Delete(txIter.Key())
	}
	//commit
	storeBatch.Write()
}

// filt related and build the fully annotated transactions.
func filterTX(b *legacy.Block, account, wallet dbm.DB) []*AnnotatedTx {
	annotatedTxs := make([]*AnnotatedTx, 0, len(b.Transactions))

	for pos, tx := range b.Transactions {
		local := false
		for _, v := range tx.Outputs {
			var hash [32]byte
			sha3pool.Sum256(hash[:], v.ControlProgram)
			key := append([]byte(accountCPPreFix), hash[:]...)

			if bytes := account.Get(key); bytes != nil {
				annotatedTxs = append(annotatedTxs, buildAnnotatedTransaction(tx, b, uint32(pos)))
				local = true
				break
			}
		}

		if local == false {
			for _, v := range tx.Inputs {
				outid, err := v.SpentOutputID()
				if err != nil {
					continue
				}
				key := append([]byte(accountUTXOPreFix), outid.Bytes()...)
				if bytes := wallet.Get(key); bytes != nil {
					annotatedTxs = append(annotatedTxs, buildAnnotatedTransaction(tx, b, uint32(pos)))
					break
				}
			}
		}
	}

	return annotatedTxs
}

func (ind *Indexer) insertAnnotatedTxs(b *legacy.Block, account, wallet dbm.DB) error {
	annotatedTxs := filterTX(b, account, wallet)
	for _, annotator := range ind.annotators {
		if err := annotator(annotatedTxs); err != nil {
			return errors.Wrap(err, "adding external annotations")
		}
	}

	storeBatch := ind.db.NewBatch()
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
