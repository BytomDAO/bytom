package protocol

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/golang/groupcache/lru"
)

var (
	maxCachedErrTxs = 1000
	maxNewTxChSize  = 1000
	// ErrTransactionNotExist is the pre-defined error message
	ErrTransactionNotExist = errors.New("transaction are not existed in the mempool")
)

// TxDesc store tx and related info for mining strategy
type TxDesc struct {
	Tx       *legacy.Tx
	Added    time.Time
	Height   uint64
	Weight   uint64
	Fee      uint64
	FeePerKB uint64
}

// TxPool is use for store the unconfirmed transaction
type TxPool struct {
	lastUpdated int64
	mtx         sync.RWMutex
	pool        map[bc.Hash]*TxDesc
	errCache    *lru.Cache
	newTxCh     chan *legacy.Tx
}

// NewTxPool init a new TxPool
func NewTxPool() *TxPool {
	return &TxPool{
		lastUpdated: time.Now().Unix(),
		pool:        make(map[bc.Hash]*TxDesc),
		errCache:    lru.New(maxCachedErrTxs),
		newTxCh:     make(chan *legacy.Tx, maxNewTxChSize),
	}
}

// GetNewTxCh return a unconfirmed transaction feed channel
func (mp *TxPool) GetNewTxCh() chan *legacy.Tx {
	return mp.newTxCh
}

// AddTransaction add a verified transaction to pool
func (mp *TxPool) AddTransaction(tx *legacy.Tx, height, fee uint64) *TxDesc {
	txD := &TxDesc{
		Tx:       tx,
		Added:    time.Now(),
		Weight:   tx.TxData.SerializedSize,
		Height:   height,
		Fee:      fee,
		FeePerKB: fee * 1000 / tx.TxHeader.SerializedSize,
	}

	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	mp.pool[tx.Tx.ID] = txD
	atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())

	mp.newTxCh <- tx
	return txD
}

// AddErrCache add a failed transaction record to lru cache
func (mp *TxPool) AddErrCache(txHash *bc.Hash, err error) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	mp.errCache.Add(txHash, err)
}

// GetErrCache return the error of the transaction
func (mp *TxPool) GetErrCache(txHash *bc.Hash) error {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	v, ok := mp.errCache.Get(txHash)
	if !ok {
		return nil
	}
	return v.(error)
}

// RemoveTransaction remove a transaction from the pool
func (mp *TxPool) RemoveTransaction(txHash *bc.Hash) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	if _, ok := mp.pool[*txHash]; ok {
		delete(mp.pool, *txHash)
		atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())
	}
}

// GetTransaction return the TxDesc by hash
func (mp *TxPool) GetTransaction(txHash *bc.Hash) (*TxDesc, error) {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	if txD, ok := mp.pool[*txHash]; ok {
		return txD, nil
	}

	return nil, ErrTransactionNotExist
}

// GetTransactions return all the transactions in the pool
func (mp *TxPool) GetTransactions() []*TxDesc {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	txDs := make([]*TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		txDs[i] = desc
		i++
	}
	return txDs
}

// IsTransactionInPool check wheather a transaction in pool or not
func (mp *TxPool) IsTransactionInPool(txHash *bc.Hash) bool {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	if _, ok := mp.pool[*txHash]; ok {
		return true
	}
	return false
}

// IsTransactionInErrCache check wheather a transaction in errCache or not
func (mp *TxPool) IsTransactionInErrCache(txHash *bc.Hash) bool {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	_, ok := mp.errCache.Get(txHash)
	return ok
}

// HaveTransaction IsTransactionInErrCache check is  transaction in errCache or pool
func (mp *TxPool) HaveTransaction(txHash *bc.Hash) bool {
	return mp.IsTransactionInPool(txHash) || mp.IsTransactionInErrCache(txHash)
}

// Count return number of transcation in pool
func (mp *TxPool) Count() int {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	count := len(mp.pool)
	return count
}
