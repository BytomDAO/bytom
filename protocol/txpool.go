package protocol

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/groupcache/lru"

	"github.com/bytom/consensus"
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
	log "github.com/sirupsen/logrus"
)

const (
	MsgNewTx = iota
	MsgRemoveTx
)

var (
	maxCachedErrTxs = 1000
	maxMsgChSize    = 1000
	maxNewTxNum     = 10000

	// ErrTransactionNotExist is the pre-defined error message
	ErrTransactionNotExist = errors.New("transaction are not existed in the mempool")
	// ErrPoolIsFull indicates the pool is full
	ErrPoolIsFull = errors.New("transaction pool reach the max number")
)

// TxDesc store tx and related info for mining strategy
type TxDesc struct {
	Tx         *types.Tx
	Added      time.Time
	StatusFail bool
	Height     uint64
	Weight     uint64
	Fee        uint64
	FeePerKB   uint64
}

type TxPoolMsg struct {
	*TxDesc
	MsgType int
}

// TxPool is use for store the unconfirmed transaction
type TxPool struct {
	lastUpdated int64
	mtx         sync.RWMutex
	pool        map[bc.Hash]*TxDesc
	utxo        map[bc.Hash]bc.Hash
	errCache    *lru.Cache
	msgCh       chan *TxPoolMsg
}

// NewTxPool init a new TxPool
func NewTxPool() *TxPool {
	return &TxPool{
		lastUpdated: time.Now().Unix(),
		pool:        make(map[bc.Hash]*TxDesc),
		utxo:        make(map[bc.Hash]bc.Hash),
		errCache:    lru.New(maxCachedErrTxs),
		msgCh:       make(chan *TxPoolMsg, maxMsgChSize),
	}
}

// GetNewTxCh return a unconfirmed transaction feed channel
func (tp *TxPool) GetMsgCh() <-chan *TxPoolMsg {
	return tp.msgCh
}

// AddTransaction add a verified transaction to pool
func (tp *TxPool) AddTransaction(tx *types.Tx, statusFail bool, height, fee uint64) (*TxDesc, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	if len(tp.pool) >= maxNewTxNum {
		return nil, ErrPoolIsFull
	}

	txD := &TxDesc{
		Tx:         tx,
		Added:      time.Now(),
		StatusFail: statusFail,
		Weight:     tx.TxData.SerializedSize,
		Height:     height,
		Fee:        fee,
		FeePerKB:   fee * 1000 / tx.TxHeader.SerializedSize,
	}

	tp.pool[tx.Tx.ID] = txD
	atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())

	for _, id := range tx.TxHeader.ResultIds {
		output, err := tx.Output(*id)
		if err != nil {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}
		if !statusFail || *output.Source.Value.AssetId == *consensus.BTMAssetID {
			tp.utxo[*id] = tx.Tx.ID
		}
	}

	tp.msgCh <- &TxPoolMsg{TxDesc: txD, MsgType: MsgNewTx}
	log.WithField("tx_id", tx.Tx.ID.String()).Debug("Add tx to mempool")
	return txD, nil
}

// AddErrCache add a failed transaction record to lru cache
func (tp *TxPool) AddErrCache(txHash *bc.Hash, err error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	tp.errCache.Add(txHash, err)
}

// GetErrCache return the error of the transaction
func (tp *TxPool) GetErrCache(txHash *bc.Hash) error {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	v, ok := tp.errCache.Get(txHash)
	if !ok {
		return nil
	}
	return v.(error)
}

// RemoveTransaction remove a transaction from the pool
func (tp *TxPool) RemoveTransaction(txHash *bc.Hash) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	txD, ok := tp.pool[*txHash]
	if !ok {
		return
	}

	for _, output := range txD.Tx.TxHeader.ResultIds {
		delete(tp.utxo, *output)
	}
	delete(tp.pool, *txHash)
	atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())

	tp.msgCh <- &TxPoolMsg{TxDesc: txD, MsgType: MsgRemoveTx}
	log.WithField("tx_id", txHash).Debug("remove tx from mempool")
}

// GetTransaction return the TxDesc by hash
func (tp *TxPool) GetTransaction(txHash *bc.Hash) (*TxDesc, error) {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	if txD, ok := tp.pool[*txHash]; ok {
		return txD, nil
	}

	return nil, ErrTransactionNotExist
}

// GetTransactions return all the transactions in the pool
func (tp *TxPool) GetTransactions() []*TxDesc {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	txDs := make([]*TxDesc, len(tp.pool))
	i := 0
	for _, desc := range tp.pool {
		txDs[i] = desc
		i++
	}
	return txDs
}

// GetTransactionUTXO return unconfirmed utxo
func (tp *TxPool) GetTransactionUTXO(tx *bc.Tx) *state.UtxoViewpoint {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	view := state.NewUtxoViewpoint()
	for _, prevout := range tx.SpentOutputIDs {
		if _, ok := tp.utxo[prevout]; ok {
			view.Entries[prevout] = storage.NewUtxoEntry(false, 0, false)
		}
	}
	return view
}

// IsTransactionInPool check wheather a transaction in pool or not
func (tp *TxPool) IsTransactionInPool(txHash *bc.Hash) bool {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	if _, ok := tp.pool[*txHash]; ok {
		return true
	}
	return false
}

// IsTransactionInErrCache check wheather a transaction in errCache or not
func (tp *TxPool) IsTransactionInErrCache(txHash *bc.Hash) bool {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	_, ok := tp.errCache.Get(txHash)
	return ok
}

// HaveTransaction IsTransactionInErrCache check is  transaction in errCache or pool
func (tp *TxPool) HaveTransaction(txHash *bc.Hash) bool {
	return tp.IsTransactionInPool(txHash) || tp.IsTransactionInErrCache(txHash)
}

// Count return number of transcation in pool
func (tp *TxPool) Count() int {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	count := len(tp.pool)
	return count
}
