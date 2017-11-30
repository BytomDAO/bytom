package query

import (
	"encoding/json"
	"fmt"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/sirupsen/logrus"
)

const (
	//TxPreFix is transactions prefix
	TxPreFix = "TXS:"
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
func (ind *Indexer) IndexTransactions(b *legacy.Block) {
	if err := ind.insertAnnotatedTxs(b); err != nil {
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

func (ind *Indexer) insertAnnotatedTxs(b *legacy.Block) error {
	var (
		annotatedTxs = make([]*AnnotatedTx, 0, len(b.Transactions))
	)

	// Build the fully annotated transactions.
	for pos, tx := range b.Transactions {
		annotatedTxs = append(annotatedTxs, buildAnnotatedTransaction(tx, b, uint32(pos)))
	}

	for _, annotator := range ind.annotators {
		if err := annotator(annotatedTxs); err != nil {
			return errors.Wrap(err, "adding external annotations")
		}
	}

	localAnnotator(annotatedTxs)

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
