package protocol

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/consensus/bcrp"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

// msg type
const (
	MsgNewTx = iota
	MsgRemoveTx
	logModule = "protocol"
)

var (
	maxCachedErrTxs = 1000
	maxMsgChSize    = 1000
	maxNewTxNum     = 10000
	maxOrphanNum    = 2000

	orphanTTL                = 10 * time.Minute
	orphanExpireScanInterval = 3 * time.Minute

	// ErrTransactionNotExist is the pre-defined error message
	ErrTransactionNotExist = errors.New("transaction are not existed in the mempool")
	// ErrPoolIsFull indicates the pool is full
	ErrPoolIsFull = errors.New("transaction pool reach the max number")
	// ErrDustTx indicates transaction is dust tx
	ErrDustTx = errors.New("transaction is dust tx")
)

var blackUtxo = map[string]bool{
	"6f083763575172bba322b149696ebd178b521057671e701500fa9e2ad69828b1": true,
	"2b60436ec77f7f28a5d4d0318286f29ec4217308837b8590990b332294ec7564": true,
	"412b2bf4b10f6b88411e00b5f6610d36fbbd992eceb93d32199931caa3610b27": true,
	"f109549a4ce3533229a0e94f84493f78f1d2103f88ce738e68951c5963b13682": true,
	"e7dc81f508b25a55e287f0e57af73ee806008de7551415f654e4875dc310ed90": true,
	"f2d5cb651bb9dcac9706daf0b544fef68bb4f07b90803038d6b86b434a98cd43": true,
	"0b9db0db5331429c26c9b0c45d3cf55fd638a140fccfd719179e223ea1c4a7f3": true,
	"df3d59e5ff9e90143e8fd15aa90c87679b03f7b6d30c48d82c3b5ca14fb85e30": true,
	"a1c584000cfa6b8ccd411bace27330cf71350f855cd3be4c6e87ff31ef3885e7": true,
	"768af8d34193b89a81dee4c10948ab2aef077f799e43c36d2a84d7bd5e185648": true,
	"9c4149f8c316b17a4d47b7403e82c73178867cb7eb6b3b653380376572add1ed": true,
	"fbb3b8e697db8b6a0ab8f75bd9be8bad10c60430186bed83a328d719e1967510": true,
	"60346fa38983c95e40a17b9be953eafe66776a548bf64516916922b2d67496f3": true,
	"a2dc7eaf1681bf926de8ed70c7239960de14eb989353072e71d2313088bdca92": true,
	"dbc3d7b47a01f71f6f1ccd575e7630d777df82e0a3961e04f0d5af62c2e91e7f": true,
	"e9cf21ac781e25e4013320407261dc697516b8873a40eba4de061d170fa4b28c": true,
	"ae8d924b60806a9f668a059e62eef75de511e39f881c90cc58ad0e8847cd69b1": true,
}

type TxMsgEvent struct{ TxMsg *TxPoolMsg }

// TxDesc store tx and related info for mining strategy
type TxDesc struct {
	Tx     *types.Tx `json:"transaction"`
	Added  time.Time `json:"-"`
	Height uint64    `json:"-"`
	Weight uint64    `json:"-"`
	Fee    uint64    `json:"-"`
}

// TxPoolMsg is use for notify pool changes
type TxPoolMsg struct {
	*TxDesc
	MsgType int
}

type orphanTx struct {
	*TxDesc
	expiration time.Time
}

// TxPool is use for store the unconfirmed transaction
type TxPool struct {
	lastUpdated     int64
	mtx             sync.RWMutex
	store           state.Store
	pool            map[bc.Hash]*TxDesc
	utxo            map[bc.Hash]*types.Tx
	orphans         map[bc.Hash]*orphanTx
	orphansByPrev   map[bc.Hash]map[bc.Hash]*orphanTx
	errCache        *lru.Cache
	eventDispatcher *event.Dispatcher
}

// NewTxPool init a new TxPool
func NewTxPool(store state.Store, dispatcher *event.Dispatcher) *TxPool {
	tp := &TxPool{
		lastUpdated:     time.Now().Unix(),
		store:           store,
		pool:            make(map[bc.Hash]*TxDesc),
		utxo:            make(map[bc.Hash]*types.Tx),
		orphans:         make(map[bc.Hash]*orphanTx),
		orphansByPrev:   make(map[bc.Hash]map[bc.Hash]*orphanTx),
		errCache:        lru.New(maxCachedErrTxs),
		eventDispatcher: dispatcher,
	}
	go tp.orphanExpireWorker()
	return tp
}

// AddErrCache add a failed transaction record to lru cache
func (tp *TxPool) AddErrCache(txHash *bc.Hash, err error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	tp.errCache.Add(txHash, err)
}

// ExpireOrphan expire all the orphans that before the input time range
func (tp *TxPool) ExpireOrphan(now time.Time) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	for hash, orphan := range tp.orphans {
		if orphan.expiration.Before(now) {
			tp.removeOrphan(&hash)
		}
	}
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

	for _, output := range txD.Tx.ResultIds {
		delete(tp.utxo, *output)
	}
	delete(tp.pool, *txHash)

	atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
	tp.eventDispatcher.Post(TxMsgEvent{TxMsg: &TxPoolMsg{TxDesc: txD, MsgType: MsgRemoveTx}})
	log.WithFields(log.Fields{"module": logModule, "tx_id": txHash}).Debug("remove tx from mempool")
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

// IsTransactionInPool check wheather a transaction in pool or not
func (tp *TxPool) IsTransactionInPool(txHash *bc.Hash) bool {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	_, ok := tp.pool[*txHash]
	return ok
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

func isTransactionNoBtmInput(tx *types.Tx) bool {
	for _, input := range tx.TxData.Inputs {
		if input.AssetID() == *consensus.BTMAssetID {
			return false
		}
	}
	return true
}

func isTransactionZeroOutput(tx *types.Tx) bool {
	for _, output := range tx.TxData.Outputs {
		if output.Amount == uint64(0) {
			return true
		}
	}
	return false
}

func isInvalidBCRPTx(tx *types.Tx) bool {
	for _, output := range tx.TxData.Outputs {
		if bcrp.IsBCRPScript(output.ControlProgram) {
			return true
		}
	}
	return false
}

func (tp *TxPool) IsDust(tx *types.Tx) bool {
	return isTransactionNoBtmInput(tx) || isTransactionZeroOutput(tx) || isInvalidBCRPTx(tx)
}

func (tp *TxPool) processTransaction(tx *types.Tx, height, fee uint64) (bool, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()

	for _, hash := range tx.SpentOutputIDs {
		if blackUtxo[hash.String()] {
			log.WithFields(log.Fields{"module": logModule, "utxo": hash.String()}).Warn("black utxo")
			return false, errors.New("black utxo")
		}
	}

	txD := &TxDesc{
		Tx:     tx,
		Weight: tx.SerializedSize,
		Height: height,
		Fee:    fee,
	}
	requireParents, err := tp.checkOrphanUtxos(tx)
	if err != nil {
		return false, err
	}

	if len(requireParents) > 0 {
		return true, tp.addOrphan(txD, requireParents)
	}

	if err := tp.addTransaction(txD); err != nil {
		return false, err
	}

	tp.processOrphans(txD)
	return false, nil
}

// ProcessTransaction is the main entry for txpool handle new tx, ignore dust tx.
func (tp *TxPool) ProcessTransaction(tx *types.Tx, height, fee uint64) (bool, error) {
	if tp.IsDust(tx) {
		log.WithFields(log.Fields{"module": logModule, "tx_id": tx.ID.String()}).Warn("dust tx")
		return false, nil
	}
	return tp.processTransaction(tx, height, fee)
}

func (tp *TxPool) addOrphan(txD *TxDesc, requireParents []*bc.Hash) error {
	if len(tp.orphans) >= maxOrphanNum {
		return ErrPoolIsFull
	}

	orphan := &orphanTx{txD, time.Now().Add(orphanTTL)}
	tp.orphans[txD.Tx.ID] = orphan
	for _, hash := range requireParents {
		if _, ok := tp.orphansByPrev[*hash]; !ok {
			tp.orphansByPrev[*hash] = make(map[bc.Hash]*orphanTx)
		}
		tp.orphansByPrev[*hash][txD.Tx.ID] = orphan
	}
	return nil
}

func (tp *TxPool) addTransaction(txD *TxDesc) error {
	if len(tp.pool) >= maxNewTxNum {
		return ErrPoolIsFull
	}

	tx := txD.Tx
	txD.Added = time.Now()
	tp.pool[tx.ID] = txD
	for _, id := range tx.ResultIds {
		_, err := tx.OriginalOutput(*id)
		if err != nil {
			// error due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		tp.utxo[*id] = tx
	}

	atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
	tp.eventDispatcher.Post(TxMsgEvent{TxMsg: &TxPoolMsg{TxDesc: txD, MsgType: MsgNewTx}})
	log.WithFields(log.Fields{"module": logModule, "tx_id": tx.ID.String()}).Debug("Add tx to mempool")
	return nil
}

func (tp *TxPool) checkOrphanUtxos(tx *types.Tx) ([]*bc.Hash, error) {
	view := state.NewUtxoViewpoint()
	if err := tp.store.GetTransactionsUtxo(view, []*bc.Tx{tx.Tx}); err != nil {
		return nil, err
	}

	hashes := []*bc.Hash{}
	for _, hash := range tx.SpentOutputIDs {
		if !view.CanSpend(&hash) && tp.utxo[hash] == nil {
			hashes = append(hashes, &hash)
		}
	}
	return hashes, nil
}

func (tp *TxPool) orphanExpireWorker() {
	ticker := time.NewTicker(orphanExpireScanInterval)
	defer ticker.Stop()

	for now := range ticker.C {
		tp.ExpireOrphan(now)
	}
}

func (tp *TxPool) processOrphans(txD *TxDesc) {
	processOrphans := []*orphanTx{}
	addRely := func(tx *types.Tx) {
		for _, outHash := range tx.ResultIds {
			orphans, ok := tp.orphansByPrev[*outHash]
			if !ok {
				continue
			}

			for _, orphan := range orphans {
				processOrphans = append(processOrphans, orphan)
			}
			delete(tp.orphansByPrev, *outHash)
		}
	}

	addRely(txD.Tx)
	for ; len(processOrphans) > 0; processOrphans = processOrphans[1:] {
		processOrphan := processOrphans[0]
		requireParents, err := tp.checkOrphanUtxos(processOrphan.Tx)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("processOrphans got unexpect error")
			continue
		}

		if len(requireParents) == 0 {
			addRely(processOrphan.Tx)
			tp.removeOrphan(&processOrphan.Tx.ID)
			tp.addTransaction(processOrphan.TxDesc)
		}
	}
}

func (tp *TxPool) removeOrphan(hash *bc.Hash) {
	orphan, ok := tp.orphans[*hash]
	if !ok {
		return
	}

	for _, spend := range orphan.Tx.SpentOutputIDs {
		orphans, ok := tp.orphansByPrev[spend]
		if !ok {
			continue
		}

		if delete(orphans, *hash); len(orphans) == 0 {
			delete(tp.orphansByPrev, spend)
		}
	}
	delete(tp.orphans, *hash)
}
