package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/config"
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
)

// utxo describes an individual account utxo.
type utxo struct {
	OutputID bc.Hash
	SourceID bc.Hash

	// Avoiding AssetAmount here so that new(utxo) doesn't produce an
	// AssetAmount with a nil AssetId.
	AssetID bc.AssetID
	Amount  uint64

	SourcePos      uint64
	ControlProgram []byte
	RefDataHash    bc.Hash

	AccountID           string
	Address             string
	ControlProgramIndex uint64
}

func NewUtxo() *utxo {
	return &utxo{}
}

func (u *utxo) source() source {
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
	UTXOs       []*utxo
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
	reserved, total, err := sourceReserver.reserve(rid, amount)
	if err != nil {
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

	if !re.checkUTXO(u) {
		return nil, errors.New("didn't find utxo")
	}

	rid := atomic.AddUint64(&re.nextReservationID, 1)
	err = re.source(u.source()).reserveUTXO(rid, u)
	if err != nil {
		return nil, err
	}

	res := &reservation{
		ID:          rid,
		Source:      u.source(),
		UTXOs:       []*utxo{u},
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
		return fmt.Errorf("couldn't find reservation %d", rid)
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

func (re *reserver) checkUTXO(u *utxo) bool {
	utxo, err := re.c.GetUtxo(&u.OutputID)
	if err != nil {
		return false
	}
	return !utxo.Spend
}

func (re *reserver) source(src source) *sourceReserver {
	re.sourcesMu.Lock()
	defer re.sourcesMu.Unlock()

	sr, ok := re.sources[src]
	if ok {
		return sr
	}

	sr = &sourceReserver{
		db:       re.db,
		src:      src,
		validFn:  re.checkUTXO,
		cached:   make(map[bc.Hash]*utxo),
		reserved: make(map[bc.Hash]uint64),
	}
	re.sources[src] = sr
	return sr
}

type sourceReserver struct {
	db       dbm.DB
	src      source
	validFn  func(u *utxo) bool
	mu       sync.Mutex
	cached   map[bc.Hash]*utxo
	reserved map[bc.Hash]uint64
}

func (sr *sourceReserver) reserve(rid uint64, amount uint64) ([]*utxo, uint64, error) {
	reservedUTXOs, reservedAmount, err := sr.reserveFromCache(rid, amount)
	if err == nil {
		return reservedUTXOs, reservedAmount, nil
	}

	// Find the set of UTXOs that match this source.
	err = sr.refillCache()
	if err != nil {
		return nil, 0, err
	}

	return sr.reserveFromCache(rid, amount)
}

func (sr *sourceReserver) reserveFromCache(rid uint64, amount uint64) ([]*utxo, uint64, error) {
	var (
		reserved, unavailable uint64
		reservedUTXOs         []*utxo
	)
	sr.mu.Lock()
	defer sr.mu.Unlock()

	for o, u := range sr.cached {
		// If the UTXO is already reserved, skip it.
		if _, ok := sr.reserved[u.OutputID]; ok {
			unavailable += u.Amount
			continue
		}
		// Cached utxos aren't guaranteed to still be valid; they may
		// have been spent. Verify that that the outputs are still in
		// the state tree.
		if !sr.validFn(u) {
			delete(sr.cached, o)
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
		return nil, 0, ErrInsufficient
	}
	if reserved < amount {
		// The account has enough for the request, but some is tied up in
		// other reservations.
		return nil, 0, ErrReserved
	}

	// We've found enough to satisfy the request.
	for _, u := range reservedUTXOs {
		sr.reserved[u.OutputID] = rid
	}

	return reservedUTXOs, reserved, nil
}

func (sr *sourceReserver) reserveUTXO(rid uint64, utxo *utxo) error {
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

func (sr *sourceReserver) refillCache() error {

	utxos, err := findMatchingUTXOs(sr.db, sr.src)
	if err != nil {
		return errors.Wrap(err)
	}

	sr.mu.Lock()
	for _, u := range utxos {
		sr.cached[u.OutputID] = u
	}
	sr.mu.Unlock()

	return nil
}

func findMatchingUTXOs(db dbm.DB, src source) ([]*utxo, error) {

	var (
		utxos       []*utxo
		accountUTXO UTXO
		rawOutputID [32]byte
		rawSourceID [32]byte
		rawRefData  [32]byte
	)

	accountUTXOIter := db.IteratorPrefix([]byte(UTXOPreFix))
	defer accountUTXOIter.Release()
	for accountUTXOIter.Next() {

		if err := json.Unmarshal(accountUTXOIter.Value(), &accountUTXO); err != nil {
			return nil, errors.Wrap(err)
		}

		if (accountUTXO.AccountID == src.AccountID) && (bytes.Equal(accountUTXO.AssetID, src.AssetID.Bytes())) {
			copy(rawOutputID[:], accountUTXO.OutputID)
			copy(rawSourceID[:], accountUTXO.SourceID)
			copy(rawRefData[:], accountUTXO.RefData)

			utxos = append(utxos, &utxo{
				OutputID:            bc.NewHash(rawOutputID),
				SourceID:            bc.NewHash(rawSourceID),
				AssetID:             src.AssetID,
				Amount:              accountUTXO.Amount,
				SourcePos:           accountUTXO.SourcePos,
				ControlProgram:      accountUTXO.Program,
				RefDataHash:         bc.NewHash(rawRefData),
				AccountID:           src.AccountID,
				Address:             accountUTXO.Address,
				ControlProgramIndex: accountUTXO.ProgramIndex,
			})

		}

	}

	if len(utxos) == 0 {
		log.WithFields(log.Fields{"AccountID": src.AccountID, "AssetID": src.AssetID.String()}).Error("can't match utxo")
		return nil, errors.New("can't match utxo")
	}

	return utxos, nil
}

func findSpecificUTXO(db dbm.DB, outHash bc.Hash) (*utxo, error) {
	u := new(utxo)
	accountUTXO := new(UTXO)

	//temp fix for coinbase UTXO isn't add to accountUTXO db, will be remove later
	if outHash == *config.GenerateGenesisTx().ResultIds[0] {
		return genesisBlockUTXO(), nil
	}

	// make sure accountUTXO existed in the db
	accountUTXOValue := db.Get(UTXOKey(outHash))
	if accountUTXOValue == nil {
		return nil, fmt.Errorf("can't find utxo: %s", outHash.String())
	}
	if err := json.Unmarshal(accountUTXOValue, &accountUTXO); err != nil {
		return nil, errors.Wrap(err)
	}

	rawOutputID := new([32]byte)
	rawAssetID := new([32]byte)
	rawSourceID := new([32]byte)
	rawRefData := new([32]byte)

	copy(rawOutputID[:], accountUTXO.OutputID)
	copy(rawAssetID[:], accountUTXO.AssetID)
	copy(rawSourceID[:], accountUTXO.SourceID)
	copy(rawRefData[:], accountUTXO.RefData)

	u.OutputID = bc.NewHash(*rawOutputID)
	u.AccountID = accountUTXO.AccountID
	u.Address = accountUTXO.Address
	u.AssetID = bc.NewAssetID(*rawAssetID)
	u.Amount = accountUTXO.Amount
	u.ControlProgramIndex = accountUTXO.ProgramIndex
	u.ControlProgram = accountUTXO.Program
	u.SourceID = bc.NewHash(*rawSourceID)
	u.SourcePos = accountUTXO.SourcePos
	u.RefDataHash = bc.NewHash(*rawRefData)

	return u, nil
}

//temp fix for coinbase UTXO isn't add to accountUTXO db, will be remove later
func genesisBlockUTXO() *utxo {
	u := new(utxo)
	tx := config.GenerateGenesisTx()

	out := tx.Outputs[0]
	resOutID := tx.ResultIds[0]
	resOut, _ := tx.Entries[*resOutID].(*bc.Output)
	log.Infof("genesis Output:%v", resOut)

	//u.AccountID =
	u.OutputID = *tx.OutputID(0)
	u.AssetID = *out.AssetId
	u.Amount = out.Amount
	u.ControlProgramIndex = 0
	u.ControlProgram = out.ControlProgram
	u.SourceID = *resOut.Source.Ref
	u.SourcePos = resOut.Source.Position
	u.RefDataHash = *resOut.Data
	return u
}
