package account

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/sync/idempotency"
)

var (
	// ErrInsufficient indicates the account doesn't contain enough
	// units of the requested asset to satisfy the reservation.
	// New units must be deposited into the account in order to
	// satisfy the request; change will not be sufficient.
	ErrInsufficient = errors.New("reservation found insufficient funds")

	// ErrReserved indicates that a reservation could not be
	// satisfied because some of the outputs were already reserved.
	// When those reservations are finalized into a transaction
	// (and no other transaction spends funds from the account),
	// new change outputs will be created
	// in sufficient amounts to satisfy the request.
	ErrReserved = errors.New("reservation found outputs already reserved")
	// ErrMatchUTXO indicates the account doesn't contain enough utxo to satisfy the reservation.
	ErrMatchUTXO = errors.New("can't match enough valid utxos")
	// ErrReservation indicates the reserver doesn't found the reservation with the provided ID.
	ErrReservation = errors.New("couldn't find reservation")
)

// UTXO describes an individual account utxo.
type UTXO struct {
	OutputID bc.Hash
	SourceID bc.Hash

	// Avoiding AssetAmount here so that new(utxo) doesn't produce an
	// AssetAmount with a nil AssetId.
	AssetID bc.AssetID
	Amount  uint64

	SourcePos      uint64
	ControlProgram []byte

	AccountID           string
	Address             string
	ControlProgramIndex uint64
	ValidHeight         uint64
}

func (u *UTXO) source() source {
	return source{AssetID: u.AssetID, AccountID: u.AccountID}
}

// source describes the criteria to use when selecting UTXOs.
type source struct {
	AssetID   bc.AssetID
	AccountID string
}

// reservation describes a reservation of a set of UTXOs belonging
// to a particular account. Reservations are immutable.
type reservation struct {
	ID          uint64
	Source      source
	UTXOs       []*UTXO
	Change      uint64
	Expiry      time.Time
	ClientToken *string
}

func newReserver(c *protocol.Chain, walletdb dbm.DB) *reserver {
	return &reserver{
		c:            c,
		db:           walletdb,
		reservations: make(map[uint64]*reservation),
		sources:      make(map[source]*sourceReserver),
	}
}

// reserver implements a utxo reserver that stores reservations
// in-memory. It relies on the account_utxos table for the source of
// truth of valid UTXOs but tracks which of those UTXOs are reserved
// in-memory.
//
// To reduce latency and prevent deadlock, no two mutexes (either on
// reserver or sourceReserver) should be held at the same time
//
// reserver ensures idempotency of reservations until the reservation
// expiration.
type reserver struct {
	c                 *protocol.Chain
	db                dbm.DB
	nextReservationID uint64
	idempotency       idempotency.Group

	reservationsMu sync.Mutex
	reservations   map[uint64]*reservation

	sourcesMu sync.Mutex
	sources   map[source]*sourceReserver
}

// Reserve selects and reserves UTXOs according to the criteria provided
// in source. The resulting reservation expires at exp.
func (re *reserver) Reserve(src source, amount uint64, clientToken *string, exp time.Time) (*reservation, error) {

	if clientToken == nil {
		return re.reserve(src, amount, clientToken, exp)
	}

	untypedRes, err := re.idempotency.Once(*clientToken, func() (interface{}, error) {
		return re.reserve(src, amount, clientToken, exp)
	})
	return untypedRes.(*reservation), err
}

func (re *reserver) reserve(src source, amount uint64, clientToken *string, exp time.Time) (res *reservation, err error) {
	sourceReserver := re.source(src)

	// Try to reserve the right amount.
	rid := atomic.AddUint64(&re.nextReservationID, 1)
	reserved, total, isImmature, err := sourceReserver.reserve(rid, amount)
	if err != nil {
		if isImmature {
			return nil, errors.WithDetail(err, "some coinbase utxos are immature")
		}
		return nil, err
	}

	res = &reservation{
		ID:          rid,
		Source:      src,
		UTXOs:       reserved,
		Expiry:      exp,
		ClientToken: clientToken,
	}

	// Save the successful reservation.
	re.reservationsMu.Lock()
	defer re.reservationsMu.Unlock()
	re.reservations[rid] = res

	// Make change if necessary
	if total > amount {
		res.Change = total - amount
	}
	return res, nil
}

// ReserveUTXO reserves a specific utxo for spending. The resulting
// reservation expires at exp.
func (re *reserver) ReserveUTXO(ctx context.Context, out bc.Hash, clientToken *string, exp time.Time) (*reservation, error) {
	if clientToken == nil {
		return re.reserveUTXO(ctx, out, exp, nil)
	}

	untypedRes, err := re.idempotency.Once(*clientToken, func() (interface{}, error) {
		return re.reserveUTXO(ctx, out, exp, clientToken)
	})
	return untypedRes.(*reservation), err
}

func (re *reserver) reserveUTXO(ctx context.Context, out bc.Hash, exp time.Time, clientToken *string) (*reservation, error) {
	u, err := findSpecificUTXO(re.db, out)
	if err != nil {
		return nil, err
	}

	//u.ValidHeight > 0 means coinbase utxo
	if u.ValidHeight > 0 && u.ValidHeight > re.c.BestBlockHeight() {
		return nil, errors.WithDetail(ErrMatchUTXO, "this coinbase utxo is immature")
	}

	rid := atomic.AddUint64(&re.nextReservationID, 1)
	err = re.source(u.source()).reserveUTXO(rid, u)
	if err != nil {
		return nil, err
	}

	res := &reservation{
		ID:          rid,
		Source:      u.source(),
		UTXOs:       []*UTXO{u},
		Expiry:      exp,
		ClientToken: clientToken,
	}
	re.reservationsMu.Lock()
	re.reservations[rid] = res
	re.reservationsMu.Unlock()
	return res, nil
}

// Cancel makes a best-effort attempt at canceling the reservation with
// the provided ID.
func (re *reserver) Cancel(ctx context.Context, rid uint64) error {
	re.reservationsMu.Lock()
	res, ok := re.reservations[rid]
	delete(re.reservations, rid)
	re.reservationsMu.Unlock()
	if !ok {
		return errors.Wrapf(ErrReservation, "rid=%d", rid)
	}
	re.source(res.Source).cancel(res)
	/*if res.ClientToken != nil {
		re.idempotency.Forget(*res.ClientToken)
	}*/
	return nil
}

// ExpireReservations cleans up all reservations that have expired,
// making their UTXOs available for reservation again.
func (re *reserver) ExpireReservations(ctx context.Context) error {
	// Remove records of any reservations that have expired.
	now := time.Now()
	var canceled []*reservation
	re.reservationsMu.Lock()
	for rid, res := range re.reservations {
		if res.Expiry.Before(now) {
			canceled = append(canceled, res)
			delete(re.reservations, rid)
		}
	}
	re.reservationsMu.Unlock()

	// If we removed any expired reservations, update the corresponding
	// source reservers.
	for _, res := range canceled {
		re.source(res.Source).cancel(res)
		/*if res.ClientToken != nil {
			re.idempotency.Forget(*res.ClientToken)
		}*/
	}

	// TODO(jackson): Cleanup any source reservers that don't have
	// anything reserved. It'll be a little tricky because of our
	// locking scheme.
	return nil
}

func (re *reserver) source(src source) *sourceReserver {
	re.sourcesMu.Lock()
	defer re.sourcesMu.Unlock()

	sr, ok := re.sources[src]
	if ok {
		return sr
	}

	sr = &sourceReserver{
		db:            re.db,
		src:           src,
		reserved:      make(map[bc.Hash]uint64),
		currentHeight: re.c.BestBlockHeight,
	}
	re.sources[src] = sr
	return sr
}

type sourceReserver struct {
	db            dbm.DB
	src           source
	currentHeight func() uint64
	mu            sync.Mutex
	reserved      map[bc.Hash]uint64
}

func (sr *sourceReserver) reserve(rid uint64, amount uint64) ([]*UTXO, uint64, bool, error) {
	var (
		reserved, unavailable uint64
		reservedUTXOs         []*UTXO
	)

	utxos, isImmature, err := findMatchingUTXOs(sr.db, sr.src, sr.currentHeight)
	if err != nil {
		return nil, 0, isImmature, errors.Wrap(err)
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()
	for _, u := range utxos {
		// If the UTXO is already reserved, skip it.
		if _, ok := sr.reserved[u.OutputID]; ok {
			unavailable += u.Amount
			continue
		}

		reserved += u.Amount
		reservedUTXOs = append(reservedUTXOs, u)
		if reserved >= amount {
			break
		}
	}
	if reserved+unavailable < amount {
		// Even if everything was available, this account wouldn't have
		// enough to satisfy the request.
		return nil, 0, isImmature, ErrInsufficient
	}
	if reserved < amount {
		// The account has enough for the request, but some is tied up in
		// other reservations.
		return nil, 0, isImmature, ErrReserved
	}

	// We've found enough to satisfy the request.
	for _, u := range reservedUTXOs {
		sr.reserved[u.OutputID] = rid
	}

	return reservedUTXOs, reserved, isImmature, nil
}

func (sr *sourceReserver) reserveUTXO(rid uint64, utxo *UTXO) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	_, isReserved := sr.reserved[utxo.OutputID]
	if isReserved {
		return ErrReserved
	}

	sr.reserved[utxo.OutputID] = rid
	return nil
}

func (sr *sourceReserver) cancel(res *reservation) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	for _, utxo := range res.UTXOs {
		delete(sr.reserved, utxo.OutputID)
	}
}

func findMatchingUTXOs(db dbm.DB, src source, currentHeight func() uint64) ([]*UTXO, bool, error) {
	utxos := []*UTXO{}
	isImmature := false
	utxoIter := db.IteratorPrefix([]byte(UTXOPreFix))
	defer utxoIter.Release()

	for utxoIter.Next() {
		u := &UTXO{}
		if err := json.Unmarshal(utxoIter.Value(), u); err != nil {
			return nil, false, errors.Wrap(err)
		}

		//u.ValidHeight > 0 means coinbase utxo
		if u.ValidHeight > 0 && u.ValidHeight > currentHeight() {
			isImmature = true
			continue
		}

		if u.AccountID == src.AccountID && u.AssetID == src.AssetID {
			utxos = append(utxos, u)
		}
	}

	if len(utxos) == 0 {
		return nil, isImmature, ErrMatchUTXO
	}
	return utxos, isImmature, nil
}

func findSpecificUTXO(db dbm.DB, outHash bc.Hash) (*UTXO, error) {
	u := &UTXO{}

	data := db.Get(StandardUTXOKey(outHash))
	if data == nil {
		if data = db.Get(ContractUTXOKey(outHash)); data == nil {
			return nil, errors.Wrapf(ErrMatchUTXO, "output_id = %s", outHash.String())
		}
	}
	return u, json.Unmarshal(data, u)
}
