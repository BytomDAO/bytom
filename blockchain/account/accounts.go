// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/vm/vmutil"
)

const (
	maxAccountCache = 1000
	aliasPrefix     = "ALI:"
	accountPrefix   = "ACC:"
	accountCPPrefix = "ACP:"
	keyNextIndex    = "NextIndex"
)

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias = errors.New("duplicate account alias")
	ErrBadIdentifier  = errors.New("either ID or alias must be specified, and not both")
)

func aliasKey(name string) []byte {
	return []byte(aliasPrefix + name)
}

//Key account store prefix
func Key(name string) []byte {
	return []byte(accountPrefix + name)
}

//CPKey account control promgram store prefix
func CPKey(hash common.Hash) []byte {
	return append([]byte(accountCPPrefix), hash[:]...)
}

// NewManager creates a new account manager
func NewManager(walletDB dbm.DB, chain *protocol.Chain) *Manager {
	var nextIndex uint64
	if index := walletDB.Get([]byte(keyNextIndex)); index != nil {
		nextIndex = uint64(binary.LittleEndian.Uint64(index))
	}
	return &Manager{
		db:           walletDB,
		chain:        chain,
		utxoDB:       newReserver(chain, walletDB),
		cache:        lru.New(maxAccountCache),
		aliasCache:   lru.New(maxAccountCache),
		delayedACPs:  make(map[*txbuilder.TemplateBuilder][]*CtrlProgram),
		acpIndexNext: nextIndex,
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
	delayedACPs   map[*txbuilder.TemplateBuilder][]*CtrlProgram

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
	Tags  map[string]interface{}
}

// Create creates a new Account.
func (m *Manager) Create(ctx context.Context, xpubs []chainkd.XPub, quorum int, alias string, tags map[string]interface{}, clientToken string) (*Account, error) {
	if existed := m.db.Get(aliasKey(alias)); existed != nil {
		return nil, fmt.Errorf("%s is an existed alias", alias)
	}

	signer, err := signers.Create(ctx, m.db, "account", xpubs, quorum, clientToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer, Alias: alias, Tags: tags}
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal account")
	}

	accountID := Key(signer.ID)
	m.db.Set(accountID, accountJSON)
	m.db.Set(aliasKey(alias), []byte(signer.ID))

	return account, nil
}

// UpdateTags modifies the tags of the specified account. The account may be
// identified either by ID or Alias, but not both.
func (m *Manager) UpdateTags(ctx context.Context, id, alias *string, tags map[string]interface{}) error {
	//TODO: use db.batch()
	if (id == nil) == (alias == nil) {
		return errors.Wrap(ErrBadIdentifier)
	}

	var accountID []byte
	if alias != nil {
		accountID = m.db.Get(aliasKey(*alias))
	} else {
		accountID = Key(*id)
	}

	accountJSON := m.db.Get(accountID)
	if accountJSON == nil {
		return errors.New("fail to find account")
	}

	var account Account
	if err := json.Unmarshal(accountJSON, &account); err != nil {
		return err
	}

	for k, v := range tags {
		switch v {
		case "":
			delete(account.Tags, k)
			m.db.Delete(aliasKey(k))
		default:
			account.Tags[k] = v
			m.db.Set(aliasKey(k), accountID)
		}
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return errors.New("failed marshal account to update tags")
	}

	m.db.Set(accountID, accountJSON)
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

	rawAccount := m.db.Get(Key(id))
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

// CreateP2PKH generate an address for the select account
func (m *Manager) CreateP2PKH(ctx context.Context, accountID string, change bool, expiresAt time.Time) (*CtrlProgram, error) {
	cp, err := m.createP2PKH(ctx, accountID, change, expiresAt)
	if err != nil {
		return nil, err
	}

	if err = m.insertAccountControlProgram(ctx, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

func (m *Manager) createP2PKH(ctx context.Context, accountID string, change bool, expiresAt time.Time) (*CtrlProgram, error) {
	account, err := m.findByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.Quorum != 1 {
		return nil, errors.New("need single key pair account to create standard transaction")
	}

	idx, err := m.nextIndex(ctx)
	if err != nil {
		return nil, err
	}
	path := signers.Path(account, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPK := derivedXPubs[0].PublicKey()
	pubHash := crypto.Ripemd160(derivedPK)

	// TODO: pass different params due to config
	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.MainNetParams)
	if err != nil {
		return nil, err
	}

	control, err := vmutil.P2PKHSigProgram([]byte(pubHash))
	if err != nil {
		return nil, err
	}

	return &CtrlProgram{
		AccountID:      account.ID,
		Address:        address.EncodeAddress(),
		KeyIndex:       idx,
		ControlProgram: control,
		Change:         change,
		ExpiresAt:      expiresAt,
	}, nil
}

func (m *Manager) createControlProgram(ctx context.Context, accountID string, change bool, expiresAt time.Time) (*CtrlProgram, error) {
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

	return &CtrlProgram{
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

//CtrlProgram is structure of account control program
type CtrlProgram struct {
	AccountID      string
	Address        string
	KeyIndex       uint64
	ControlProgram []byte
	Change         bool
	ExpiresAt      time.Time
}

func (m *Manager) insertAccountControlProgram(ctx context.Context, progs ...*CtrlProgram) error {
	var hash common.Hash
	for _, prog := range progs {
		accountCP, err := json.Marshal(prog)
		if err != nil {
			return err
		}

		sha3pool.Sum256(hash[:], prog.ControlProgram)
		m.db.Set(CPKey(hash), accountCP)
	}
	return nil
}

// GetCoinbaseControlProgram will return a coinbase script
func (m *Manager) GetCoinbaseControlProgram(height uint64) ([]byte, error) {
	signerIter := m.db.IteratorPrefix([]byte(accountPrefix))
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

	err = m.insertAccountControlProgram(ctx, &CtrlProgram{
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

func saveIndex(db dbm.DB, index uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, index)
	db.Set([]byte(keyNextIndex), buf)
}

func (m *Manager) nextIndex(ctx context.Context) (uint64, error) {
	m.acpMu.Lock()
	defer m.acpMu.Unlock()

	n := m.acpIndexNext
	m.acpIndexNext++
	saveIndex(m.db, m.acpIndexNext)
	return n, nil
}

// QueryAll will return all the account in the db
func (m *Manager) QueryAll(ctx context.Context) (interface{}, error) {
	accounts := make([]interface{}, 0)
	accountIter := m.db.IteratorPrefix([]byte(accountPrefix))
	for accountIter.Next() {
		accounts = append(accounts, string(accountIter.Value()))
	}
	return accounts, nil
}
