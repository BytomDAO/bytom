// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"context"
	"encoding/binary"
	"encoding/json"
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
	ErrFindAccount    = errors.New("fail to find account")
	ErrMarshalAccount = errors.New("failed marshal account")
	ErrMarshalTags    = errors.New("failed marshal account to update tags")
	ErrStandardQuorum = errors.New("need single key pair account to create standard transaction")
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
	Tags  map[string]interface{} `json:"tags,omitempty"`
}

// Create creates a new Account.
func (m *Manager) Create(ctx context.Context, xpubs []chainkd.XPub, quorum int, alias string, tags map[string]interface{}, accessToken string) (*Account, error) {
	if existed := m.db.Get(aliasKey(alias)); existed != nil {
		return nil, ErrDuplicateAlias
	}

	signer, err := signers.Create(ctx, m.db, "account", xpubs, quorum, accessToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer, Alias: alias, Tags: tags}
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return nil, ErrMarshalAccount
	}
	storeBatch := m.db.NewBatch()

	accountID := Key(signer.ID)
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

	rawAccount := m.db.Get(Key(accountID))
	if rawAccount == nil {
		return ErrFindAccount
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

	rawAccount, err := json.Marshal(account)
	if err != nil {
		return ErrMarshalTags
	}

	m.db.Set(Key(accountID), rawAccount)
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
		return nil, ErrFindAccount
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
		return nil, ErrFindAccount
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

func (m *Manager) GetAliasByID(id string) string {
	var account Account

	rawAccount := m.db.Get(Key(id))
	if rawAccount == nil {
		log.Warn("fail to find account")
		return ""
	}

	if err := json.Unmarshal(rawAccount, &account); err != nil {
		log.Warn(err)
		return ""
	}

	return account.Alias
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
		return nil, ErrStandardQuorum
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

// DeleteAccount deletes the account's ID or alias matching accountInfo.
func (m *Manager) DeleteAccount(in struct {
	AccountInfo string `json:"account_info"`
}) error {

	account := Account{}
	storeBatch := m.db.NewBatch()

	accountID := in.AccountInfo
	if s, err := m.FindByAlias(nil, in.AccountInfo); err == nil {
		accountID = s.ID
	}

	rawAccount := m.db.Get(Key(accountID))
	if rawAccount == nil {
		return nil
	}
	if err := json.Unmarshal(rawAccount, &account); err != nil {
		return err
	}

	storeBatch.Delete(aliasKey(account.Alias))
	storeBatch.Delete(Key(account.ID))
	storeBatch.Write()

	return nil
}

type annotatedAccount struct {
	Alias    string           `json:"alias"`
	ID       string           `json:"id"`
	Quorum   int              `json:"quorum"`
	KeyIndex uint64           `json:"key_index"`
	XPubs    []chainkd.XPub   `json:"xpubs"`
	Tags     *json.RawMessage `json:"tags"`
}

// ListAccounts will return the accounts in the db
func (m *Manager) ListAccounts(id string) ([]annotatedAccount, error) {
	account := Account{}
	tmpAccount := annotatedAccount{}
	accounts := make([]annotatedAccount, 0)
	jsonTags := json.RawMessage(`{}`)

	accountIter := m.db.IteratorPrefix([]byte(accountPrefix + id))
	defer accountIter.Release()

	for accountIter.Next() {
		if err := json.Unmarshal(accountIter.Value(), &account); err != nil {
			return nil, err
		}

		tmpAccount.Alias = account.Alias
		tmpAccount.ID = account.ID
		tmpAccount.Quorum = account.Quorum
		tmpAccount.KeyIndex = account.KeyIndex
		tmpAccount.XPubs = account.XPubs
		if account.Tags != nil {
			t, err := json.Marshal(account.Tags)
			if err != nil {
				return nil, err
			}
			jsonTags = t
		}
		tmpAccount.Tags = &jsonTags

		accounts = append(accounts, tmpAccount)
	}

	return accounts, nil
}
