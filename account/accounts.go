// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"sort"
	"strings"
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
	indexPrefix     = "ACIDX:"
)

var miningAddressKey = []byte("miningAddress")

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias = errors.New("duplicate account alias")
	ErrFindAccount    = errors.New("fail to find account")
	ErrMarshalAccount = errors.New("failed marshal account")
	ErrMarshalTags    = errors.New("failed marshal account to update tags")
)

func aliasKey(name string) []byte {
	return []byte(aliasPrefix + name)
}

func indexKeys(xpubs []chainkd.XPub) []byte {
	xpubStrings := make([]string, len(xpubs))
	for i, xpub := range xpubs {
		xpubStrings[i] = xpub.String()
	}
	sort.Strings(xpubStrings)
	suffix := strings.Join(xpubStrings, "")

	return []byte(indexPrefix + suffix)
}

//Key account store prefix
func Key(name string) []byte {
	return []byte(accountPrefix + name)
}

//CPKey account control promgram store prefix
func CPKey(hash common.Hash) []byte {
	return append([]byte(accountCPPrefix), hash[:]...)
}

func convertUnit64ToBytes(nextIndex uint64) []byte {
	buf := make([]byte, 8)
	binary.PutUvarint(buf, nextIndex)
	return buf
}

func convertBytesToUint64(rawIndex []byte) uint64 {
	result, _ := binary.Uvarint(rawIndex)
	return result
}

// NewManager creates a new account manager
func NewManager(walletDB dbm.DB, chain *protocol.Chain) *Manager {
	return &Manager{
		db:          walletDB,
		chain:       chain,
		utxoDB:      newReserver(chain, walletDB),
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*CtrlProgram),
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

	accIndexMu  sync.Mutex
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
	ID    string
	Alias string
	Tags  map[string]interface{}
}

func (m *Manager) getNextXpubsIndex(xpubs []chainkd.XPub) uint64 {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	var nextIndex uint64 = 1
	if rawIndexBytes := m.db.Get(indexKeys(xpubs)); rawIndexBytes != nil {
		nextIndex = convertBytesToUint64(rawIndexBytes) + 1
	}

	m.db.Set(indexKeys(xpubs), convertUnit64ToBytes(nextIndex))

	return nextIndex
}

// Create creates a new Account.
func (m *Manager) Create(ctx context.Context, xpubs []chainkd.XPub, quorum int, alias string, tags map[string]interface{}) (*Account, error) {
	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if existed := m.db.Get(aliasKey(normalizedAlias)); existed != nil {
		return nil, ErrDuplicateAlias
	}

	signer, err := signers.Create("account", xpubs, quorum, m.getNextXpubsIndex(xpubs))
	id := signers.IDGenerate()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer, ID: id, Alias: normalizedAlias, Tags: tags}
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return nil, ErrMarshalAccount
	}
	storeBatch := m.db.NewBatch()

	accountID := Key(id)
	storeBatch.Set(accountID, rawAccount)
	storeBatch.Set(aliasKey(alias), []byte(id))
	storeBatch.Write()

	return account, nil
}

// UpdateTags modifies the tags of the specified account. The account may be
// identified either by ID or Alias, but not both.
func (m *Manager) UpdateTags(ctx context.Context, accountInfo string, tags map[string]interface{}) (err error) {
	account := &Account{}
	if account, err = m.FindByAlias(nil, accountInfo); err != nil {
		if account, err = m.findByID(ctx, accountInfo); err != nil {
			return err
		}
	}

	account.Tags = tags
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return ErrMarshalTags
	}

	m.db.Set(Key(account.ID), rawAccount)
	m.cacheMu.Lock()
	m.cache.Add(account.ID, account)
	m.cacheMu.Unlock()
	return nil
}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(ctx context.Context, alias string) (*Account, error) {
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
func (m *Manager) findByID(ctx context.Context, id string) (*Account, error) {
	m.cacheMu.Lock()
	cachedAccount, ok := m.cache.Get(id)
	m.cacheMu.Unlock()
	if ok {
		return cachedAccount.(*Account), nil
	}

	rawAccount := m.db.Get(Key(id))
	if rawAccount == nil {
		return nil, ErrFindAccount
	}

	account := &Account{}
	if err := json.Unmarshal(rawAccount, account); err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.cache.Add(id, account)
	m.cacheMu.Unlock()
	return account, nil
}

// GetAliasByID return the account alias by given ID
func (m *Manager) GetAliasByID(id string) string {
	account := &Account{}

	rawAccount := m.db.Get(Key(id))
	if rawAccount == nil {
		log.Warn("fail to find account")
		return ""
	}

	if err := json.Unmarshal(rawAccount, account); err != nil {
		log.Warn(err)
		return ""
	}
	return account.Alias
}

// CreateAddress generate an address for the select account
func (m *Manager) CreateAddress(ctx context.Context, accountID string) (cp *CtrlProgram, err error) {
	account, err := m.findByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return m.createAddress(ctx, account)
}

// CreateAddress generate an address for the select account
func (m *Manager) createAddress(ctx context.Context, account *Account) (cp *CtrlProgram, err error) {
	if len(account.XPubs) == 1 {
		cp, err = m.createP2PKH(ctx, account)
	} else {
		cp, err = m.createP2SH(ctx, account)
	}
	if err != nil {
		return nil, err
	}

	if err = m.insertAccountControlProgram(ctx, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

func (m *Manager) createP2PKH(ctx context.Context, account *Account) (*CtrlProgram, error) {
	idx := m.getNextXpubsIndex(account.Signer.XPubs)
	path := signers.Path(account.Signer, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPK := derivedXPubs[0].PublicKey()
	pubHash := crypto.Ripemd160(derivedPK)

	// TODO: pass different params due to config
	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.MainNetParams)
	if err != nil {
		return nil, err
	}

	control, err := vmutil.P2WPKHProgram([]byte(pubHash))
	if err != nil {
		return nil, err
	}

	return &CtrlProgram{
		AccountID:      account.ID,
		Address:        address.EncodeAddress(),
		KeyIndex:       idx,
		ControlProgram: control,
	}, nil
}

func (m *Manager) createP2SH(ctx context.Context, account *Account) (*CtrlProgram, error) {
	idx := m.getNextXpubsIndex(account.Signer.XPubs)
	path := signers.Path(account.Signer, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	signScript, err := vmutil.P2SPMultiSigProgram(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}
	scriptHash := crypto.Sha256(signScript)

	// TODO: pass different params due to config
	address, err := common.NewAddressWitnessScriptHash(scriptHash, &consensus.MainNetParams)
	if err != nil {
		return nil, err
	}

	control, err := vmutil.P2WSHProgram(scriptHash)
	if err != nil {
		return nil, err
	}

	return &CtrlProgram{
		AccountID:      account.ID,
		Address:        address.EncodeAddress(),
		KeyIndex:       idx,
		ControlProgram: control,
	}, nil
}

//CtrlProgram is structure of account control program
type CtrlProgram struct {
	AccountID      string
	Address        string
	KeyIndex       uint64
	ControlProgram []byte
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

// IsLocalControlProgram check is the input control program belong to local
func (m *Manager) IsLocalControlProgram(prog []byte) bool {
	var hash common.Hash
	sha3pool.Sum256(hash[:], prog)
	bytes := m.db.Get(CPKey(hash))
	return bytes != nil
}

// GetCoinbaseControlProgram will return a coinbase script
func (m *Manager) GetCoinbaseControlProgram() ([]byte, error) {
	if data := m.db.Get(miningAddressKey); data != nil {
		cp := &CtrlProgram{}
		return cp.ControlProgram, json.Unmarshal(data, cp)
	}

	accountIter := m.db.IteratorPrefix([]byte(accountPrefix))
	defer accountIter.Release()
	if !accountIter.Next() {
		log.Warningf("GetCoinbaseControlProgram: can't find any account in db")
		return vmutil.DefaultCoinbaseProgram()
	}

	account := &Account{}
	if err := json.Unmarshal(accountIter.Value(), account); err != nil {
		return nil, err
	}

	program, err := m.createAddress(nil, account)
	if err != nil {
		return nil, err
	}

	rawCP, err := json.Marshal(program)
	if err != nil {
		return nil, err
	}

	m.db.Set(miningAddressKey, rawCP)
	return program.ControlProgram, nil
}

// DeleteAccount deletes the account's ID or alias matching accountInfo.
func (m *Manager) DeleteAccount(in struct {
	AccountInfo string `json:"account_info"`
}) (err error) {
	account := &Account{}
	if account, err = m.FindByAlias(nil, in.AccountInfo); err != nil {
		if account, err = m.findByID(nil, in.AccountInfo); err != nil {
			return err
		}
	}

	storeBatch := m.db.NewBatch()

	m.cacheMu.Lock()
	m.aliasCache.Remove(account.Alias)
	m.cacheMu.Unlock()

	storeBatch.Delete(aliasKey(account.Alias))
	storeBatch.Delete(Key(account.ID))
	storeBatch.Write()

	return nil
}

// ListAccounts will return the accounts in the db
func (m *Manager) ListAccounts(id string) ([]*Account, error) {
	accounts := []*Account{}
	accountIter := m.db.IteratorPrefix([]byte(accountPrefix + id))
	defer accountIter.Release()

	for accountIter.Next() {
		account := &Account{}
		if err := json.Unmarshal(accountIter.Value(), &account); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// ListControlProgram return all the local control program
func (m *Manager) ListControlProgram() ([]*CtrlProgram, error) {
	var cps []*CtrlProgram
	cpIter := m.db.IteratorPrefix([]byte(accountCPPrefix))
	defer cpIter.Release()

	for cpIter.Next() {
		cp := &CtrlProgram{}
		if err := json.Unmarshal(cpIter.Value(), cp); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}

	return cps, nil
}
