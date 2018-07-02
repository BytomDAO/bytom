package account

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
)

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

func newUtxoKeeper(chain *protocol.Chain, walletdb dbm.DB) *utxoKeeper {
	uk := &utxoKeeper{
		db:            walletdb,
		currentHeight: chain.BestBlockHeight,
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

func (uk *utxoKeeper) RemoveUnconfirmedUtxo(hashes []*bc.Hash) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	for _, hash := range hashes {
		delete(uk.unconfirmed, *hash)
	}
}

// Cancel makes a best-effort attempt at canceling the reservation with the provided ID.
func (uk *utxoKeeper) Cancel(rid uint64) {
	uk.mtx.Lock()
	uk.cancel(rid)
	uk.mtx.Unlock()
}

func (uk *utxoKeeper) Reserve(accountID string, assetID *bc.AssetID, amount uint64, useUnconfirmed bool, exp time.Time) (*reservation, error) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	utxos, immatureAmount := uk.findUTXOs(accountID, assetID, useUnconfirmed)
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
		change: amount - optAmount,
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

	u, err := uk.findUTXO(outHash, useUnconfirmed)
	if err != nil {
		return nil, err
	}

	if u.ValidHeight > uk.currentHeight() {
		return nil, errors.WithDetail(ErrMatchUTXO, "this coinbase utxo is immature")
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
		uk.mtx.Lock()
		for rid, res := range uk.reservations {
			if res.expiry.Before(now) {
				uk.cancel(rid)
			}
		}
		uk.mtx.Unlock()
	}
}

func (uk *utxoKeeper) optUTXOs(utxos []*UTXO, amount uint64) ([]*UTXO, uint64, uint64) {
	var optAmount, reservedAmount uint64
	optUtxos := []*UTXO{}
	for _, u := range utxos {
		if _, ok := uk.reserved[u.OutputID]; ok {
			reservedAmount += u.Amount
			continue
		}

		optAmount += u.Amount
		optUtxos = append(optUtxos, u)
		if optAmount >= amount {
			break
		}
	}
	return optUtxos, optAmount, reservedAmount
}

func (uk *utxoKeeper) findUTXOs(accountID string, assetID *bc.AssetID, useUnconfirmed bool) ([]*UTXO, uint64) {
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
			log.WithField("err", err).Error("utxoKeeper findUTXOs fail on unmarshal utxo")
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

func (uk *utxoKeeper) findUTXO(outHash bc.Hash, useUnconfirmed bool) (*UTXO, error) {
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
	return nil, errors.Wrapf(ErrMatchUTXO, "output_id = %s", outHash.String())
}
