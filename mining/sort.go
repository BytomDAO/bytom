package mining

import (
	"sort"
	"time"

	"github.com/bytom/protocol/bc/types"
)

// A TxInfo is a record of a tx, recording gas & fee information
type TxInfo struct {
	tx          *types.Tx
	gasUsed     uint64
	isGasOnlyTx bool
	fee         uint64
	timestamp   time.Time
}

func gasUsedLessThan(tx1Info, txInfo2 *TxInfo) bool {
	return tx1Info.gasUsed < txInfo2.gasUsed
}

func timeLessThan(tx1Info, txInfo2 *TxInfo) bool {
	return tx1Info.timestamp.Unix() < txInfo2.timestamp.Unix()
}

type lessFunc func(txInfo1, txInfo2 *TxInfo) bool

// multiSorter implements the Sort interface, sorting the txInfos within.
type multiSorter struct {
	txInfos []TxInfo
	less    []lessFunc
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *multiSorter) Sort(txInfos []TxInfo) {
	ms.txInfos = txInfos
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *multiSorter) Len() int {
	return len(ms.txInfos)
}

// Swap is part of sort.Interface.
func (ms *multiSorter) Swap(i, j int) {
	ms.txInfos[i], ms.txInfos[j] = ms.txInfos[j], ms.txInfos[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other). Note that it can call the
// less functions twice per call. We could change the functions to return
// -1, 0, 1 and reduce the number of calls for greater efficiency: an
// exercise for the reader.
func (ms *multiSorter) Less(i, j int) bool {
	p, q := &ms.txInfos[i], &ms.txInfos[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}
