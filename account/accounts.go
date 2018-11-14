// Package account stores and tracks accounts within a Bytom Core.
package account

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm/vmutil"
)

const (
	maxAccountCache  = 1000
	HardenedKeyStart = 0x80000000
)

var (
	accountIndexPrefix  = []byte("AccountIndex:")
	accountPrefix       = []byte("Account:")
	aliasPrefix         = []byte("AccountAlias:")
	contractIndexPrefix = []byte("ContractIndex")
	contractPrefix      = []byte("Contract:")
	miningAddressKey    = []byte("MiningAddress")
	CoinbaseAbKey       = []byte("CoinbaseArbitrary")
)

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias  = errors.New("duplicate account alias")
	ErrFindAccount     = errors.New("fail to find account")
	ErrMarshalAccount  = errors.New("failed marshal account")
	ErrInvalidAddress  = errors.New("invalid address")
	ErrFindCtrlProgram = errors.New("fail to find account control program")
	ErrDeriveRule      = errors.New("invalid key derive rule")
	ErrContractIndex   = errors.New("exceed the maximum addresses per account")
	ErrAccountIndex    = errors.New("exceed the maximum accounts per xpub")
)

// ContractKey account control promgram store prefix
func ContractKey(hash common.Hash) []byte {
	return append(contractPrefix, hash[:]...)
}

// Key account store prefix
func Key(name string) []byte {
	return append(accountPrefix, []byte(name)...)
}

func aliasKey(name string) []byte {
	return append(aliasPrefix, []byte(name)...)
}

func bip44ContractIndexKey(accountID string, change bool) []byte {
	key := append(contractIndexPrefix, accountID...)
	if change {
		return append(key, []byte{1}...)
	}
	return append(key, []byte{0}...)
}

func contractIndexKey(accountID string) []byte {
	return append(contractIndexPrefix, []byte(accountID)...)
}

// Account is structure of Bytom account
type Account struct {
	*signers.Signer
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

//CtrlProgram is structure of account control program
type CtrlProgram struct {
	AccountID      string
	Address        string
	KeyIndex       uint64
	ControlProgram []byte
	Change         bool // Mark whether this control program is for UTXO change
}

// Manager stores accounts and their associated control programs.
type Manager struct {
	db         dbm.DB
	chain      *protocol.Chain
	utxoKeeper *utxoKeeper

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	delayedACPsMu sync.Mutex
	delayedACPs   map[*txbuilder.TemplateBuilder][]*CtrlProgram

	accIndexMu sync.Mutex
	accountMu  sync.Mutex
}

// NewManager creates a new account manager
func NewManager(walletDB dbm.DB, chain *protocol.Chain) *Manager {
	return &Manager{
		db:          walletDB,
		chain:       chain,
		utxoKeeper:  newUtxoKeeper(chain.BestBlockHeight, walletDB),
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*CtrlProgram),
	}
}

// AddUnconfirmedUtxo add untxo list to utxoKeeper
func (m *Manager) AddUnconfirmedUtxo(utxos []*UTXO) {
	m.utxoKeeper.AddUnconfirmedUtxo(utxos)
}

// Create creates a new Account.
func (m *Manager) Create(xpubs []chainkd.XPub, quorum int, alias string) (*Account, error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if existed := m.db.Get(aliasKey(normalizedAlias)); existed != nil {
		return nil, ErrDuplicateAlias
	}

	index := uint64(1)
	if rawIndexBytes := m.db.Get(GetAccountIndexKey(xpubs)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes) + 1
	}
	if index >= HardenedKeyStart {
		return nil, ErrAccountIndex
	}

	signer, err := signers.Create("account", xpubs, quorum, index, signers.BIP0044)
	id := signers.IDGenerate()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer, ID: id, Alias: normalizedAlias}
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return nil, ErrMarshalAccount
	}

	accountID := Key(id)
	storeBatch := m.db.NewBatch()
	storeBatch.Set(GetAccountIndexKey(xpubs), common.Unit64ToBytes(index))
	storeBatch.Set(accountID, rawAccount)
	storeBatch.Set(aliasKey(normalizedAlias), []byte(id))
	storeBatch.Write()
	return account, nil
}

func (m *Manager) UpdateAccountAlias(accountID string, newAlias string) (err error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	account, err := m.FindByID(accountID)
	if err != nil {
		return err
	}
	oldAlias := account.Alias

	normalizedAlias := strings.ToLower(strings.TrimSpace(newAlias))
	if existed := m.db.Get(aliasKey(normalizedAlias)); existed != nil {
		return ErrDuplicateAlias
	}

	m.cacheMu.Lock()
	m.aliasCache.Remove(oldAlias)
	m.cacheMu.Unlock()

	account.Alias = normalizedAlias
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return ErrMarshalAccount
	}

	storeBatch := m.db.NewBatch()
	storeBatch.Delete(aliasKey(oldAlias))
	storeBatch.Set(Key(accountID), rawAccount)
	storeBatch.Set(aliasKey(normalizedAlias), []byte(accountID))
	storeBatch.Write()
	return nil
}

// CreateAddress generate an address for the select account
func (m *Manager) CreateAddress(accountID string, change bool) (cp *CtrlProgram, err error) {
	account, err := m.FindByID(accountID)
	if err != nil {
		return nil, err
	}
	return m.createAddress(account, change)
}

// DeleteAccount deletes the account's ID or alias matching accountInfo.
func (m *Manager) DeleteAccount(aliasOrID string) (err error) {
	account := &Account{}
	if account, err = m.FindByAlias(aliasOrID); err != nil {
		if account, err = m.FindByID(aliasOrID); err != nil {
			return err
		}
	}

	m.cacheMu.Lock()
	m.aliasCache.Remove(account.Alias)
	m.cacheMu.Unlock()

	storeBatch := m.db.NewBatch()
	storeBatch.Delete(aliasKey(account.Alias))
	storeBatch.Delete(Key(account.ID))
	storeBatch.Write()
	return nil
}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(alias string) (*Account, error) {
	m.cacheMu.Lock()
	cachedID, ok := m.aliasCache.Get(alias)
	m.cacheMu.Unlock()
	if ok {
		return m.FindByID(cachedID.(string))
	}

	rawID := m.db.Get(aliasKey(alias))
	if rawID == nil {
		return nil, ErrFindAccount
	}

	accountID := string(rawID)
	m.cacheMu.Lock()
	m.aliasCache.Add(alias, accountID)
	m.cacheMu.Unlock()
	return m.FindByID(accountID)
}

// FindByID returns an account's Signer record by its ID.
func (m *Manager) FindByID(id string) (*Account, error) {
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

// GetAccountByProgram return Account by given CtrlProgram
func (m *Manager) GetAccountByProgram(program *CtrlProgram) (*Account, error) {
	rawAccount := m.db.Get(Key(program.AccountID))
	if rawAccount == nil {
		return nil, ErrFindAccount
	}

	account := &Account{}
	return account, json.Unmarshal(rawAccount, account)
}

// GetAliasByID return the account alias by given ID
func (m *Manager) GetAliasByID(id string) string {
	rawAccount := m.db.Get(Key(id))
	if rawAccount == nil {
		log.Warn("GetAliasByID fail to find account")
		return ""
	}

	account := &Account{}
	if err := json.Unmarshal(rawAccount, account); err != nil {
		log.Warn(err)
	}
	return account.Alias
}

func (m *Manager) GetCoinbaseArbitrary() []byte {
	if arbitrary := m.db.Get(CoinbaseAbKey); arbitrary != nil {
		return arbitrary
	}
	return []byte{}
}

// GetCoinbaseControlProgram will return a coinbase script
func (m *Manager) GetCoinbaseControlProgram() ([]byte, error) {
	cp, err := m.GetCoinbaseCtrlProgram()
	if err == ErrFindAccount {
		log.Warningf("GetCoinbaseControlProgram: can't find any account in db")
		return vmutil.DefaultCoinbaseProgram()
	}
	if err != nil {
		return nil, err
	}
	return cp.ControlProgram, nil
}

// GetCoinbaseCtrlProgram will return the coinbase CtrlProgram
func (m *Manager) GetCoinbaseCtrlProgram() (*CtrlProgram, error) {
	if data := m.db.Get(miningAddressKey); data != nil {
		cp := &CtrlProgram{}
		return cp, json.Unmarshal(data, cp)
	}

	accountIter := m.db.IteratorPrefix([]byte(accountPrefix))
	defer accountIter.Release()
	if !accountIter.Next() {
		return nil, ErrFindAccount
	}

	account := &Account{}
	if err := json.Unmarshal(accountIter.Value(), account); err != nil {
		return nil, err
	}

	program, err := m.createAddress(account, false)
	if err != nil {
		return nil, err
	}

	rawCP, err := json.Marshal(program)
	if err != nil {
		return nil, err
	}

	m.db.Set(miningAddressKey, rawCP)
	return program, nil
}

// GetContractIndex return the current index
func (m *Manager) GetContractIndex(accountID string) uint64 {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	index := uint64(0)
	if rawIndexBytes := m.db.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetBip44ContractIndex return the current bip44 contract index
func (m *Manager) GetBip44ContractIndex(accountID string, change bool) uint64 {
	index := uint64(0)
	if rawIndexBytes := m.db.Get(bip44ContractIndexKey(accountID, change)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetLocalCtrlProgramByAddress return CtrlProgram by given address
func (m *Manager) GetLocalCtrlProgramByAddress(address string) (*CtrlProgram, error) {
	program, err := m.getProgramByAddress(address)
	if err != nil {
		return nil, err
	}

	var hash [32]byte
	sha3pool.Sum256(hash[:], program)
	rawProgram := m.db.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, ErrFindCtrlProgram
	}

	cp := &CtrlProgram{}
	return cp, json.Unmarshal(rawProgram, cp)
}

// GetMiningAddress will return the mining address
func (m *Manager) GetMiningAddress() (string, error) {
	cp, err := m.GetCoinbaseCtrlProgram()
	if err != nil {
		return "", err
	}
	return cp.Address, nil
}

// IsLocalControlProgram check is the input control program belong to local
func (m *Manager) IsLocalControlProgram(prog []byte) bool {
	var hash common.Hash
	sha3pool.Sum256(hash[:], prog)
	bytes := m.db.Get(ContractKey(hash))
	return bytes != nil
}

// ListAccounts will return the accounts in the db
func (m *Manager) ListAccounts(id string) ([]*Account, error) {
	accounts := []*Account{}
	accountIter := m.db.IteratorPrefix(Key(strings.TrimSpace(id)))
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
	cps := []*CtrlProgram{}
	cpIter := m.db.IteratorPrefix(contractPrefix)
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

func (m *Manager) ListUnconfirmedUtxo(accountID string, isSmartContract bool) []*UTXO {
	utxos := m.utxoKeeper.ListUnconfirmed()
	result := []*UTXO{}
	for _, utxo := range utxos {
		if segwit.IsP2WScript(utxo.ControlProgram) != isSmartContract && (accountID == utxo.AccountID || accountID == "") {
			result = append(result, utxo)
		}
	}
	return result
}

// RemoveUnconfirmedUtxo remove utxos from the utxoKeeper
func (m *Manager) RemoveUnconfirmedUtxo(hashes []*bc.Hash) {
	m.utxoKeeper.RemoveUnconfirmedUtxo(hashes)
}

// SetMiningAddress will set the mining address
func (m *Manager) SetMiningAddress(miningAddress string) (string, error) {
	program, err := m.getProgramByAddress(miningAddress)
	if err != nil {
		return "", err
	}

	cp := &CtrlProgram{
		Address:        miningAddress,
		ControlProgram: program,
	}
	rawCP, err := json.Marshal(cp)
	if err != nil {
		return "", err
	}

	m.db.Set(miningAddressKey, rawCP)
	return m.GetMiningAddress()
}

func (m *Manager) SetCoinbaseArbitrary(arbitrary []byte) {
	m.db.Set(CoinbaseAbKey, arbitrary)
}

// CreateAddress generate an address for the select account
func (m *Manager) createAddress(account *Account, change bool) (cp *CtrlProgram, err error) {
	addrIdx, err := m.getNextContractIndex(account, change)
	if err != nil {
		return nil, err
	}

	path, err := signers.Path(account.Signer, signers.AccountKeySpace, change, addrIdx)
	if err != nil {
		return nil, err
	}

	if len(account.XPubs) == 1 {
		cp, err = m.createP2PKH(account, path)
	} else {
		cp, err = m.createP2SH(account, path)
	}
	if err != nil {
		return nil, err
	}
	cp.KeyIndex, cp.Change = addrIdx, change
	return cp, m.insertControlPrograms(cp)
}

func (m *Manager) createP2PKH(account *Account, path [][]byte) (*CtrlProgram, error) {
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPK := derivedXPubs[0].PublicKey()
	pubHash := crypto.Ripemd160(derivedPK)

	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
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
		ControlProgram: control,
	}, nil
}

func (m *Manager) createP2SH(account *Account, path [][]byte) (*CtrlProgram, error) {
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	signScript, err := vmutil.P2SPMultiSigProgram(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}
	scriptHash := crypto.Sha256(signScript)

	address, err := common.NewAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
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
		ControlProgram: control,
	}, nil
}

func GetAccountIndexKey(xpubs []chainkd.XPub) []byte {
	var hash [32]byte
	var xPubs []byte
	cpy := append([]chainkd.XPub{}, xpubs[:]...)
	sort.Sort(signers.SortKeys(cpy))
	for _, xpub := range cpy {
		xPubs = append(xPubs, xpub[:]...)
	}
	sha3pool.Sum256(hash[:], xPubs)
	return append(accountIndexPrefix, hash[:]...)
}

func (m *Manager) getNextBip32ContractIndex(accountID string, change bool) (uint64, error) {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	nextIndex := uint64(1)
	if rawIndexBytes := m.db.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		nextIndex = common.BytesToUnit64(rawIndexBytes) + 1
	}
	if nextIndex >= HardenedKeyStart {
		return 0, ErrContractIndex
	}

	m.db.Set(contractIndexKey(accountID), common.Unit64ToBytes(nextIndex))
	return nextIndex, nil
}

func (m *Manager) getNextBip44ContractIndex(accountID string, change bool) (uint64, error) {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	nextIndex := uint64(1)
	if rawIndexBytes := m.db.Get(bip44ContractIndexKey(accountID, change)); rawIndexBytes != nil {
		nextIndex = common.BytesToUnit64(rawIndexBytes) + 1
	}
	if nextIndex >= HardenedKeyStart {
		return 0, ErrContractIndex
	}
	m.db.Set(bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(nextIndex))
	return nextIndex, nil
}

func (m *Manager) getNextContractIndex(account *Account, change bool) (uint64, error) {
	switch account.DeriveRule {
	case signers.BIP0032:
		return m.getNextBip32ContractIndex(account.ID, change)
	case signers.BIP0044:
		return m.getNextBip44ContractIndex(account.ID, change)
	}
	return 0, ErrDeriveRule
}

func (m *Manager) getProgramByAddress(address string) ([]byte, error) {
	addr, err := common.DecodeAddress(address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	redeemContract := addr.ScriptAddress()
	program := []byte{}
	switch addr.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return nil, ErrInvalidAddress
	}
	if err != nil {
		return nil, err
	}
	return program, nil
}

func (m *Manager) insertControlPrograms(progs ...*CtrlProgram) error {
	var hash common.Hash
	for _, prog := range progs {
		accountCP, err := json.Marshal(prog)
		if err != nil {
			return err
		}

		sha3pool.Sum256(hash[:], prog.ControlProgram)
		m.db.Set(ContractKey(hash), accountCP)
	}
	return nil
}
