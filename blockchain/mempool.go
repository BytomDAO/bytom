package blockchain

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/groupcache/lru"

	"github.com/bytom/protocol/bc"
)

var (
	maxCachedErrTxs = 1000

	ErrTransactionNotExist = errors.New("transaction are not existed in the mempool")
)

type TxDesc struct {
	Tx       *bc.Tx
	Added    time.Time
	Height   uint64
	Fee      uint64
	FeePerKB uint64
}

type TxPool struct {
	lastUpdated int64
	mtx         sync.RWMutex
	pool        map[bc.Hash]*TxDesc
	errCache    *lru.Cache
}

func NewTxPool() *TxPool {
	return &TxPool{
		lastUpdated: time.Now().Unix(),
		pool:        make(map[bc.Hash]*TxDesc),
		errCache:    lru.New(maxCachedErrTxs),
	}
}

func (mp *TxPool) AddTransaction(tx *bc.Tx, height uint64, fee uint64) *TxDesc {
	txD := &TxDesc{
		Tx:       tx,
		Added:    time.Now(),
		Height:   height,
		Fee:      fee,
		FeePerKB: fee * 1000 / tx.TxHeader.SerializedSize,
	}

	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	mp.pool[tx.ID] = txD
	atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())
	return txD
}

func (mp *TxPool) AddErrCache(txHash *bc.Hash) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	mp.errCache.Add(txHash, nil)
}

func (mp *TxPool) removeTransaction(txHash *bc.Hash) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	if _, ok := mp.pool[*txHash]; ok {
		delete(mp.pool, *txHash)
		atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())
	}
}

func (mp *TxPool) GetTransaction(txHash *bc.Hash) (*TxDesc, error) {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	if txD, ok := mp.pool[*txHash]; ok {
		return txD, nil
	}

	return nil, ErrTransactionNotExist
}

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

func (mp *TxPool) IsTransactionInPool(txHash *bc.Hash) bool {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	if _, ok := mp.pool[*txHash]; ok {
		return true
	}
	return false
}

func (mp *TxPool) IsTransactionInErrCache(txHash *bc.Hash) bool {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	_, ok := mp.errCache.Get(txHash)
	return ok
}

func (mp *TxPool) HaveTransaction(txHash *bc.Hash) bool {
	return mp.IsTransactionInPool(txHash) || mp.IsTransactionInErrCache(txHash)
}

func (mp *TxPool) Count() int {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	count := len(mp.pool)
	return count
}
