package account

import (
	"container/list"
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

const desireUtxoCount = 5

// pre-define error types
var (
	ErrInsufficient = errors.New("reservation found insufficient funds")
	ErrImmature     = errors.New("reservation found immature funds")
	ErrReserved     = errors.New("reservation found outputs already reserved")
	ErrMatchUTXO    = errors.New("can't find utxo with given hash")
	ErrReservation  = errors.New("couldn't find reservation")
)

// UTXO describes an individual account utxo.
type UTXO struct {
	OutputID            bc.Hash
	SourceID            bc.Hash
	AssetID             bc.AssetID
	Amount              uint64
	SourcePos           uint64
	ControlProgram      []byte
	AccountID           string
	Address             string
	ControlProgramIndex uint64
	ValidHeight         uint64
	Change              bool
}

// reservation describes a reservation of a set of UTXOs
type reservation struct {
	id     uint64
	utxos  []*UTXO
	change uint64
	expiry time.Time
}

type utxoKeeper struct {
	// `sync/atomic` expects the first word in an allocated struct to be 64-bit
	// aligned on both ARM and x86-32. See https://goo.gl/zW7dgq for more details.
	nextIndex     uint64
	db            dbm.DB
	mtx           sync.RWMutex
	currentHeight func() uint64

	unconfirmed  map[bc.Hash]*UTXO
	reserved     map[bc.Hash]uint64
	reservations map[uint64]*reservation
}

func newUtxoKeeper(f func() uint64, walletdb dbm.DB) *utxoKeeper {
	uk := &utxoKeeper{
		db:            walletdb,
		currentHeight: f,
		unconfirmed:   make(map[bc.Hash]*UTXO),
		reserved:      make(map[bc.Hash]uint64),
		reservations:  make(map[uint64]*reservation),
	}
	go uk.expireWorker()
	return uk
}

func (uk *utxoKeeper) AddUnconfirmedUtxo(utxos []*UTXO) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	for _, utxo := range utxos {
		uk.unconfirmed[utxo.OutputID] = utxo
	}
}

// Cancel canceling the reservation with the provided ID.
func (uk *utxoKeeper) Cancel(rid uint64) {
	uk.mtx.Lock()
	uk.cancel(rid)
	uk.mtx.Unlock()
}

// ListUnconfirmed return all the unconfirmed utxos
func (uk *utxoKeeper) ListUnconfirmed() []*UTXO {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	utxos := []*UTXO{}
	for _, utxo := range uk.unconfirmed {
		utxos = append(utxos, utxo)
	}
	return utxos
}

func (uk *utxoKeeper) RemoveUnconfirmedUtxo(hashes []*bc.Hash) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	for _, hash := range hashes {
		delete(uk.unconfirmed, *hash)
	}
}

func (uk *utxoKeeper) Reserve(accountID string, assetID *bc.AssetID, amount uint64, useUnconfirmed bool, exp time.Time) (*reservation, error) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	utxos, immatureAmount := uk.findUtxos(accountID, assetID, useUnconfirmed)
	optUtxos, optAmount, reservedAmount := uk.optUTXOs(utxos, amount)
	if optAmount+reservedAmount+immatureAmount < amount {
		return nil, ErrInsufficient
	}

	if optAmount+reservedAmount < amount {
		return nil, ErrImmature
	}

	if optAmount < amount {
		return nil, ErrReserved
	}

	result := &reservation{
		id:     atomic.AddUint64(&uk.nextIndex, 1),
		utxos:  optUtxos,
		change: optAmount - amount,
		expiry: exp,
	}

	uk.reservations[result.id] = result
	for _, u := range optUtxos {
		uk.reserved[u.OutputID] = result.id
	}
	return result, nil
}

func (uk *utxoKeeper) ReserveParticular(outHash bc.Hash, useUnconfirmed bool, exp time.Time) (*reservation, error) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	if _, ok := uk.reserved[outHash]; ok {
		return nil, ErrReserved
	}

	u, err := uk.findUtxo(outHash, useUnconfirmed)
	if err != nil {
		return nil, err
	}

	if u.ValidHeight > uk.currentHeight() {
		return nil, ErrImmature
	}

	result := &reservation{
		id:     atomic.AddUint64(&uk.nextIndex, 1),
		utxos:  []*UTXO{u},
		expiry: exp,
	}
	uk.reservations[result.id] = result
	uk.reserved[u.OutputID] = result.id
	return result, nil
}

func (uk *utxoKeeper) cancel(rid uint64) {
	res, ok := uk.reservations[rid]
	if !ok {
		return
	}

	delete(uk.reservations, rid)
	for _, utxo := range res.utxos {
		delete(uk.reserved, utxo.OutputID)
	}
}

func (uk *utxoKeeper) expireWorker() {
	ticker := time.NewTicker(1000 * time.Millisecond)
	for now := range ticker.C {
		uk.expireReservation(now)
	}
}
func (uk *utxoKeeper) expireReservation(t time.Time) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	for rid, res := range uk.reservations {
		if res.expiry.Before(t) {
			uk.cancel(rid)
		}
	}
}

func (uk *utxoKeeper) findUtxos(accountID string, assetID *bc.AssetID, useUnconfirmed bool) ([]*UTXO, uint64) {
	immatureAmount := uint64(0)
	currentHeight := uk.currentHeight()
	utxos := []*UTXO{}
	appendUtxo := func(u *UTXO) {
		if u.AccountID != accountID || u.AssetID != *assetID {
			return
		}
		if u.ValidHeight > currentHeight {
			immatureAmount += u.Amount
		} else {
			utxos = append(utxos, u)
		}
	}

	utxoIter := uk.db.IteratorPrefix([]byte(UTXOPreFix))
	defer utxoIter.Release()
	for utxoIter.Next() {
		u := &UTXO{}
		if err := json.Unmarshal(utxoIter.Value(), u); err != nil {
			log.WithField("err", err).Error("utxoKeeper findUtxos fail on unmarshal utxo")
			continue
		}
		appendUtxo(u)
	}
	if !useUnconfirmed {
		return utxos, immatureAmount
	}

	for _, u := range uk.unconfirmed {
		appendUtxo(u)
	}
	return utxos, immatureAmount
}

func (uk *utxoKeeper) findUtxo(outHash bc.Hash, useUnconfirmed bool) (*UTXO, error) {
	if u, ok := uk.unconfirmed[outHash]; useUnconfirmed && ok {
		return u, nil
	}

	u := &UTXO{}
	if data := uk.db.Get(StandardUTXOKey(outHash)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	if data := uk.db.Get(ContractUTXOKey(outHash)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	return nil, ErrMatchUTXO
}

func (uk *utxoKeeper) optUTXOs(utxos []*UTXO, amount uint64) ([]*UTXO, uint64, uint64) {
	//sort the utxo by amount, bigger amount in front
	var optAmount, reservedAmount uint64
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Amount > utxos[j].Amount
	})

	//push all the available utxos into list
	utxoList := list.New()
	for _, u := range utxos {
		if _, ok := uk.reserved[u.OutputID]; ok {
			reservedAmount += u.Amount
			continue
		}
		utxoList.PushBack(u)
	}

	optList := list.New()
	for node := utxoList.Front(); node != nil; node = node.Next() {
		//append utxo if we haven't reached the required amount
		if optAmount < amount {
			optList.PushBack(node.Value)
			optAmount += node.Value.(*UTXO).Amount
			continue
		}

		largestNode := optList.Front()
		replaceList := list.New()
		replaceAmount := optAmount - largestNode.Value.(*UTXO).Amount

		for ; node != nil && replaceList.Len() <= desireUtxoCount-optList.Len(); node = node.Next() {
			replaceList.PushBack(node.Value)
			if replaceAmount += node.Value.(*UTXO).Amount; replaceAmount >= amount {
				optList.Remove(largestNode)
				optList.PushBackList(replaceList)
				optAmount = replaceAmount
				break
			}
		}

		//largestNode remaining the same means that there is nothing to be replaced
		if largestNode == optList.Front() {
			break
		}
	}

	optUtxos := []*UTXO{}
	for e := optList.Front(); e != nil; e = e.Next() {
		optUtxos = append(optUtxos, e.Value.(*UTXO))
	}
	return optUtxos, optAmount, reservedAmount
}
