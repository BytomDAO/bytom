// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bytom/blockchain/pin"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/log"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/vm/vmutil"
	"github.com/golang/groupcache/lru"

	dbm "github.com/tendermint/tmlibs/db"
)

const maxAccountCache = 1000

var (
	ErrDuplicateAlias = errors.New("duplicate account alias")
	ErrBadIdentifier  = errors.New("either ID or alias must be specified, and not both")
)

func NewManager(db dbm.DB, chain *protocol.Chain, pinStore *pin.Store) *Manager {
	return &Manager{
		db:          db,
		chain:       chain,
		utxoDB:      newReserver(db, chain),
		pinStore:    pinStore,
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*controlProgram),
	}
}

// Manager stores accounts and their associated control programs.
type Manager struct {
	db       dbm.DB
	chain    *protocol.Chain
	utxoDB   *reserver
	indexer  Saver
	pinStore *pin.Store

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	delayedACPsMu sync.Mutex
	delayedACPs   map[*txbuilder.TemplateBuilder][]*controlProgram

	acpMu        sync.Mutex
	acpIndexNext uint64 // next acp index in our block
	acpIndexCap  uint64 // points to end of block
}

func (m *Manager) IndexAccounts(indexer Saver) {
	m.indexer = indexer
}

// ExpireReservations removes reservations that have expired periodically.
// It blocks until the context is canceled.
func (m *Manager) ExpireReservations(ctx context.Context, period time.Duration) {
	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			log.Printf(ctx, "Deposed, ExpireReservations exiting")
			return
		case <-ticks:
			err := m.utxoDB.ExpireReservations(ctx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}

type Account struct {
	*signers.Signer
	Alias string
	Tags  map[string]interface{}
	Address  common.Address
}

// Create creates a new Account.
func (m *Manager) Create(ctx context.Context, xpubs []chainkd.XPub, quorum int, alias string, tags map[string]interface{}, clientToken string) (*Account, error) {
	//if ret := m.db.Get([]byte(alias));ret != nil {
	//return nil,errors.New("alias already exists")
	//}

	accountSigner, err := signers.Create(ctx, m.db, "account", xpubs, quorum, clientToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account_id := json.RawMessage(accountSigner.ID)
	account := &Account{
		Signer: accountSigner,
		Alias:  alias,
		Tags:   tags,
	}

	acc, err := json.Marshal(account)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal account")
	}
	if len(acc) > 0 {
		m.db.Set(account_id, acc)
		m.db.Set(json.RawMessage("ali"+alias), account_id)
	}

	err = m.indexAnnotatedAccount(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated account")
	}

	return account, nil
}

// UpdateTags modifies the tags of the specified account. The account may be
// identified either by ID or Alias, but not both.
func (m *Manager) UpdateTags(ctx context.Context, id, alias *string, tags map[string]interface{}) error {

	if (id == nil) == (alias == nil) {
		return errors.Wrap(ErrBadIdentifier)
	}

	var key_id []byte
	if alias != nil {
		key_id = m.db.Get([]byte(*alias))
	} else {
		key_id = json.RawMessage(*id)
	}

	bytes := m.db.Get(key_id)
	if bytes == nil {
		return errors.New("no exit this account.")
	}

	var account Account
	err := json.Unmarshal(bytes, &account)
	if err != nil {
		return errors.New("this account can't be unmarshal.")
	}

	for k, v := range tags {
		switch v {
		case "":
			delete(account.Tags, k)
		default:
			account.Tags[k] = v
		}

	}

	acc, err := json.Marshal(account)
	if err != nil {

		return errors.New("failed marshal account to update tags")

	} else if len(acc) == 0 {

		return errors.New("failed update account tags")

	} else {

		m.db.Set(key_id, acc)
		return nil
	}

}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(ctx context.Context, alias string) (*signers.Signer, error) {
	var accountID string

	m.cacheMu.Lock()
	cachedID, ok := m.aliasCache.Get(alias)
	m.cacheMu.Unlock()
	if ok {
		accountID = cachedID.(string)
	} else {
		/*const q = `SELECT account_id FROM accounts WHERE alias=$1`
		err := m.db.QueryRowContext(ctx, q, alias).Scan(&accountID)
		if err == stdsql.ErrNoRows {
			return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "alias: %s", alias)
		}
		if err != nil {
			return nil, errors.Wrap(err)
		}*/
		bytez := m.db.Get([]byte(fmt.Sprintf("alias_account:%v", alias)))
		accountID = string(bytez[:])
		m.cacheMu.Lock()
		m.aliasCache.Add(alias, accountID)
		m.cacheMu.Unlock()
	}
	return m.findByID(ctx, accountID)
}

// findByID returns an account's Signer record by its ID.
func (m *Manager) findByID(ctx context.Context, id string) (*signers.Signer, error) {
	m.cacheMu.Lock()
	cached, ok := m.cache.Get(id)
	m.cacheMu.Unlock()
	if ok {
		return cached.(*signers.Signer), nil
	}

	bytes := m.db.Get(json.RawMessage(id))
	if bytes == nil {
		return nil, errors.New("not find this account.")
	}

	var account Account
	err := json.Unmarshal(bytes, &account)
	if err != nil {
		return nil, errors.New("failed unmarshal this account.")
	}

	m.cacheMu.Lock()
	m.cache.Add(id, account.Signer)
	m.cacheMu.Unlock()
	return account.Signer, nil
}

type controlProgram struct {
	accountID      string
	keyIndex       uint64
	controlProgram []byte
	change         bool
	expiresAt      time.Time
}

func (m *Manager) createControlProgram(ctx context.Context, accountID string, change bool, expiresAt time.Time) (*controlProgram, error) {
	account, err := m.findByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	idx, err := m.nextIndex(ctx)
	if err != nil {
		return nil, err
	}

	path := signers.Path(account, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	control, err := vmutil.P2SPMultiSigProgram(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}
	return &controlProgram{
		accountID:      account.ID,
		keyIndex:       idx,
		controlProgram: control,
		change:         change,
		expiresAt:      expiresAt,
	}, nil
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func (m *Manager) CreateControlProgram(ctx context.Context, accountID string, change bool, expiresAt time.Time) ([]byte, error) {
	cp, err := m.createControlProgram(ctx, accountID, change, expiresAt)
	if err != nil {
		return nil, err
	}
	err = m.insertAccountControlProgram(ctx, cp)
	if err != nil {
		return nil, err
	}
	return cp.controlProgram, nil
}

type ControlProgram struct {
	AccountID      string
	KeyIndex       uint64
	ControlProgram []byte
	Change         bool
	ExpiresAt      time.Time
}

func (m *Manager) insertAccountControlProgram(ctx context.Context, progs ...*controlProgram) error {

	var b32 [32]byte
	for _, p := range progs {

		acp, err := json.Marshal(&struct {
			AccountID      string
			KeyIndex       uint64
			ControlProgram []byte
			Change         bool
			ExpiresAt      time.Time
		}{
			AccountID:      p.accountID,
			KeyIndex:       p.keyIndex,
			ControlProgram: p.controlProgram,
			Change:         p.change,
			ExpiresAt:      p.expiresAt})

		if err != nil {
			return errors.Wrap(err, "failed marshal controlProgram")
		}
		if len(acp) > 0 {
			sha3pool.Sum256(b32[:], p.controlProgram)
			m.db.Set(json.RawMessage("acp"+string(b32[:])), acp)
		}
	}

	return errors.Wrap(nil)
}

func (m *Manager) nextIndex(ctx context.Context) (uint64, error) {
	m.acpMu.Lock()
	defer m.acpMu.Unlock()

	if m.acpIndexNext >= m.acpIndexCap {

		const incrby = 10000 // start 1,increments by 10,000
		if m.acpIndexCap <= incrby {
			m.acpIndexCap = incrby + 1
		} else {
			m.acpIndexCap += incrby
		}
		m.acpIndexNext = m.acpIndexCap - incrby
	}

	n := m.acpIndexNext
	m.acpIndexNext++
	return n, nil
}

func (m *Manager) QueryAll(ctx context.Context) (interface{}, error) {
	ret := make([]interface{}, 0)

	iter := m.db.Iterator()
	for iter.Next() {
		key := string(iter.Key())
		if key[:3] != "acc" {
			continue
		}
		ret = append(ret, string(iter.Value()))
	}

	return ret, nil
}
