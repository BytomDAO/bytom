// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/vm/vmutil"
)

const (
	maxAccountCache = 1000
	aliasPreFix     = "ALI:"
	accountPreFix   = "ACC:"
	accountCPPreFix = "ACP:"
)

func aliasKey(name string) []byte {
	return []byte(aliasPreFix + name)
}

func accountKey(name string) []byte {
	return []byte(accountPreFix + name)
}

func accountCPKey(hash [32]byte) []byte {
	return append([]byte(accountCPPreFix), hash[:]...)
}

// NewManager creates a new account manager
func NewManager(db, walletDB dbm.DB, chain *protocol.Chain) *Manager {
	return &Manager{
		db:          db,
		chain:       chain,
		utxoDB:      newReserver(chain, walletDB),
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*controlProgram),
	}
}

// Manager stores accounts and their associated control programs.
type Manager struct {
	db     dbm.DB
	chain  *protocol.Chain
	utxoDB *reserver

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	delayedACPsMu sync.Mutex
	delayedACPs   map[*txbuilder.TemplateBuilder][]*controlProgram

	acpMu        sync.Mutex
	acpIndexNext uint64 // next acp index in our block
	acpIndexCap  uint64 // points to end of block
}

// ExpireReservations removes reservations that have expired periodically.
// It blocks until the context is canceled.
func (m *Manager) ExpireReservations(ctx context.Context, period time.Duration) {
	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			log.Info("Deposed, ExpireReservations exiting")
			return
		case <-ticks:
			err := m.utxoDB.ExpireReservations(ctx)
			if err != nil {
				log.WithField("error", err).Error("Expire reservations")
			}
		}
	}
}

// Account is structure of Bytom account
type Account struct {
	*signers.Signer
	Alias string
	Tags  map[string]interface{} `json:tags,omitempty`
}

// Create creates a new Account.
func (m *Manager) Create(ctx context.Context, xpubs []chainkd.XPub, quorum int, alias string, tags map[string]interface{}, accessToken string) (*Account, error) {
	if existed := m.db.Get(aliasKey(alias)); existed != nil {
		return nil, fmt.Errorf("%s is an duplicated account alias", alias)
	}

	signer, err := signers.Create(ctx, m.db, "account", xpubs, quorum, accessToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer, Alias: alias, Tags: tags}
	rawAccount, err := json.MarshalIndent(account, "", " ")
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal account")
	}

	storeBatch := m.db.NewBatch()

	accountID := accountKey(signer.ID)
	storeBatch.Set(accountID, rawAccount)
	storeBatch.Set(aliasKey(alias), []byte(signer.ID))
	storeBatch.Write()

	return account, nil
}

// UpdateTags modifies the tags of the specified account. The account may be
// identified either by ID or Alias, but not both.
func (m *Manager) UpdateTags(ctx context.Context, accountInfo string, tags map[string]interface{}) error {
	var account Account

	accountID := accountInfo
	if s, err := m.FindByAlias(nil, accountInfo); err == nil {
		accountID = s.ID
	}

	rawAccount := m.db.Get(accountKey(accountID))
	if rawAccount == nil {
		return errors.New("fail to find account")
	}
	if err := json.Unmarshal(rawAccount, &account); err != nil {
		return err
	}

	for k, v := range tags {
		switch v {
		case "":
			delete(account.Tags, k)
		default:
			if account.Tags == nil {
				account.Tags = make(map[string]interface{})
			}
			account.Tags[k] = v
		}
	}

	rawAccount, err := json.MarshalIndent(account, "", " ")
	if err != nil {
		return errors.New("failed marshal account to update tags")
	}

	m.db.Set(accountKey(accountID), rawAccount)
	return nil
}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(ctx context.Context, alias string) (*signers.Signer, error) {
	m.cacheMu.Lock()
	cachedID, ok := m.aliasCache.Get(alias)
	m.cacheMu.Unlock()
	if ok {
		return m.findByID(ctx, cachedID.(string))
	}

	rawID := m.db.Get(aliasKey(alias))
	if rawID == nil {
		return nil, errors.New("fail to find account by alias")
	}

	accountID := string(rawID)
	m.cacheMu.Lock()
	m.aliasCache.Add(alias, accountID)
	m.cacheMu.Unlock()
	return m.findByID(ctx, accountID)
}

// findByID returns an account's Signer record by its ID.
func (m *Manager) findByID(ctx context.Context, id string) (*signers.Signer, error) {
	m.cacheMu.Lock()
	cachedSigner, ok := m.cache.Get(id)
	m.cacheMu.Unlock()
	if ok {
		return cachedSigner.(*signers.Signer), nil
	}

	rawAccount := m.db.Get(accountKey(id))
	if rawAccount == nil {
		return nil, errors.New("fail to find account")
	}

	var account Account
	if err := json.Unmarshal(rawAccount, &account); err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.cache.Add(id, account.Signer)
	m.cacheMu.Unlock()
	return account.Signer, nil
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
		AccountID:      account.ID,
		KeyIndex:       idx,
		ControlProgram: control,
		Change:         change,
		ExpiresAt:      expiresAt,
	}, nil
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func (m *Manager) CreateControlProgram(ctx context.Context, accountID string, change bool, expiresAt time.Time) ([]byte, error) {
	cp, err := m.createControlProgram(ctx, accountID, change, expiresAt)
	if err != nil {
		return nil, err
	}

	if err = m.insertAccountControlProgram(ctx, cp); err != nil {
		return nil, err
	}
	return cp.ControlProgram, nil
}

type controlProgram struct {
	AccountID      string
	KeyIndex       uint64
	ControlProgram []byte
	Change         bool
	ExpiresAt      time.Time
}

func (m *Manager) insertAccountControlProgram(ctx context.Context, progs ...*controlProgram) error {
	var hash [32]byte
	for _, prog := range progs {
		accountCP, err := json.Marshal(prog)
		if err != nil {
			return err
		}

		sha3pool.Sum256(hash[:], prog.ControlProgram)
		m.db.Set(accountCPKey(hash), accountCP)
	}
	return nil
}

// GetCoinbaseControlProgram will return a coinbase script
func (m *Manager) GetCoinbaseControlProgram(height uint64) ([]byte, error) {
	signerIter := m.db.IteratorPrefix([]byte(accountPreFix))
	if !signerIter.Next() {
		log.Warningf("GetCoinbaseControlProgram: can't find any account in db")
		return vmutil.CoinbaseProgram(nil, 0, height)
	}
	rawSigner := signerIter.Value()
	signerIter.Release()

	signer := &signers.Signer{}
	if err := json.Unmarshal(rawSigner, signer); err != nil {
		log.Errorf("GetCoinbaseControlProgram: fail to unmarshal signer %v", err)
		return vmutil.CoinbaseProgram(nil, 0, height)
	}

	ctx := context.Background()
	idx, err := m.nextIndex(ctx)
	if err != nil {
		log.Errorf("GetCoinbaseControlProgram: fail to get nextIndex %v", err)
		return vmutil.CoinbaseProgram(nil, 0, height)
	}
	path := signers.Path(signer, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)

	script, err := vmutil.CoinbaseProgram(derivedPKs, signer.Quorum, height)
	if err != nil {
		return script, err
	}

	err = m.insertAccountControlProgram(ctx, &controlProgram{
		AccountID:      signer.ID,
		KeyIndex:       idx,
		ControlProgram: script,
		Change:         false,
	})
	if err != nil {
		log.Errorf("GetCoinbaseControlProgram: fail to insertAccountControlProgram %v", err)
	}
	return script, nil
}

func (m *Manager) nextIndex(ctx context.Context) (uint64, error) {
	m.acpMu.Lock()
	defer m.acpMu.Unlock()

	//TODO: fix this part, really serious security breach
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

func (m *Manager) DeleteAccount(accountInfo string) error {

	account := Account{}
	storeBatch := m.db.NewBatch()

	accountID := accountInfo
	if s, err := m.FindByAlias(nil, accountInfo); err == nil {
		accountID = s.ID
	}

	rawAccount := m.db.Get(accountKey(accountID))
	if rawAccount == nil {
		return nil
	}
	if err := json.Unmarshal(rawAccount, &account); err != nil {
		return err
	}

	storeBatch.Delete(aliasKey(account.Alias))
	storeBatch.Delete(accountKey(account.ID))
	storeBatch.Write()

	return nil
}

// ListAccounts will return the accounts in the db
func (m *Manager) ListAccounts(after string, limit, defaultLimit int) ([]string, string, bool, error) {

	var (
		zafter int
		err    error
		last   bool
	)

	if after != "" {
		zafter, err = strconv.Atoi(after)
		if err != nil {
			return nil, "", false, errors.WithDetailf(errors.New("Invalid after"), "value: %q", zafter)
		}
	}

	accounts := make([]string, 0)
	accountIter := m.db.IteratorPrefix([]byte(accountPreFix))
	defer accountIter.Release()

	for accountIter.Next() {
		accounts = append(accounts, string(accountIter.Value()))
	}

	start, end := 0, len(accounts)

	if len(accounts) == 0 {
		return nil, "", true, errors.New("No accounts")
	} else if len(accounts) > zafter {
		start = zafter
	} else {
		return nil, "", false, errors.WithDetailf(errors.New("Invalid after"), "value: %q", zafter)
	}

	if len(accounts) > zafter+limit {
		end = zafter + limit
	}

	if len(accounts) == end || len(accounts) < defaultLimit {
		last = true
	}

	return accounts[start:end], strconv.Itoa(end), last, nil
}
