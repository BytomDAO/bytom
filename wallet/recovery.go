package wallet

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	dbm "github.com/bytom/database/leveldb"
)

const (
	// acctRecoveryWindow defines the account derivation lookahead used when
	// attempting to recover the set of used accounts.
	acctRecoveryWindow = uint64(6)

	// addrRecoveryWindow defines the address derivation lookahead used when
	// attempting to recover the set of used addresses.
	addrRecoveryWindow = uint64(128)
)

//recoveryKey key for db store recovery info.
var (
	recoveryKey = []byte("RecoveryInfo")

	// ErrRecoveryBusy another recovery in progress, can not get recovery manager lock
	ErrRecoveryBusy = errors.New("another recovery in progress")

	// ErrInvalidAcctID can not find account by account id
	ErrInvalidAcctID = errors.New("invalid account id")
)

// branchRecoveryState maintains the required state in-order to properly
// recover addresses derived from a particular account's internal or external
// derivation branch.
//
// A branch recovery state supports operations for:
//  - Expanding the look-ahead horizon based on which indexes have been found.
//  - Registering derived addresses with indexes within the horizon.
//  - Reporting an invalid child index that falls into the horizon.
//  - Reporting that an address has been found.
//  - Retrieving all currently derived addresses for the branch.
//  - Looking up a particular address by its child index.
type branchRecoveryState struct {
	// recoveryWindow defines the key-derivation lookahead used when
	// attempting to recover the set of addresses on this branch.
	RecoveryWindow uint64

	// horizion records the highest child index watched by this branch.
	Horizon uint64

	// nextUnfound maintains the child index of the successor to the highest
	// index that has been found during recovery of this branch.
	NextUnfound uint64
}

// newBranchRecoveryState creates a new branchRecoveryState that can be used to
// track either the external or internal branch of an account's derivation path.
func newBranchRecoveryState(recoveryWindow uint64) *branchRecoveryState {
	return &branchRecoveryState{
		RecoveryWindow: recoveryWindow,
		Horizon:        1,
		NextUnfound:    1,
	}
}

// extendHorizon returns the current horizon and the number of addresses that
// must be derived in order to maintain the desired recovery window.
func (brs *branchRecoveryState) extendHorizon() (uint64, uint64) {
	// Compute the new horizon, which should surpass our last found address
	// by the recovery window.
	curHorizon := brs.Horizon

	minValidHorizon := brs.NextUnfound + brs.RecoveryWindow

	// If the current horizon is sufficient, we will not have to derive any
	// new keys.
	if curHorizon >= minValidHorizon {
		return curHorizon, 0
	}

	// Otherwise, the number of addresses we should derive corresponds to
	// the delta of the two horizons, and we update our new horizon.
	delta := minValidHorizon - curHorizon
	brs.Horizon = minValidHorizon

	return curHorizon, delta
}

// reportFound updates the last found index if the reported index exceeds the
// current value.
func (brs *branchRecoveryState) reportFound(index uint64) {
	if index >= brs.NextUnfound {
		brs.NextUnfound = index + 1
	}
}

// addressRecoveryState is used to manage the recovery of addresses generated
// under a particular BIP32/BIP44 account. Each account tracks both an external and
// internal branch recovery state, both of which use the same recovery window.
type addressRecoveryState struct {
	// ExternalBranch is the recovery state of addresses generated for
	// external use, i.e. receiving addresses.
	ExternalBranch *branchRecoveryState

	// InternalBranch is the recovery state of addresses generated for
	// internal use, i.e. change addresses.
	InternalBranch *branchRecoveryState

	Account *account.Account
}

func newAddressRecoveryState(recoveryWindow uint64, account *account.Account) *addressRecoveryState {
	return &addressRecoveryState{
		ExternalBranch: newBranchRecoveryState(recoveryWindow),
		InternalBranch: newBranchRecoveryState(recoveryWindow),
		Account:        account,
	}
}

// recoveryState used to record the status of a recovery process.
type recoveryState struct {
	// XPubs recovery account xPubs
	XPubs []chainkd.XPub

	// The time to start the recovery task, used to detemine whether
	// recovery task is completed.
	StartTime time.Time

	// XPubsStatus maintains a map of each requested XPub to its active
	// account recovery state.
	XPubsStatus *branchRecoveryState

	// AcctStatus maintains a map of each requested key scope to its active
	// recovery state.
	AccountsStatus map[string]*addressRecoveryState
}

func newRecoveryState() *recoveryState {
	return &recoveryState{
		AccountsStatus: make(map[string]*addressRecoveryState),
		StartTime:      time.Now(),
	}
}

// stateForScope returns a ScopeRecoveryState for the provided key scope. If one
// does not already exist, a new one will be generated with the RecoveryState's
// recoveryWindow.
func (rs *recoveryState) stateForScope(account *account.Account) {
	// If the account recovery state already exists, return it.
	if _, ok := rs.AccountsStatus[account.ID]; ok {
		return
	}

	// Otherwise, initialize the recovery state for this scope with the
	// chosen recovery window.
	rs.AccountsStatus[account.ID] = newAddressRecoveryState(addrRecoveryWindow, account)
}

// recoveryManager manage recovery wallet from key.
type recoveryManager struct {
	mu sync.Mutex

	db         dbm.DB
	accountMgr *account.Manager

	locked int32

	started bool

	// state encapsulates and allocates the necessary recovery state for all
	// key scopes and subsidiary derivation paths.
	state *recoveryState

	//addresses all addresses derivation lookahead used when
	// attempting to recover the set of used addresses.
	addresses map[bc.Hash]*account.CtrlProgram
}

// newRecoveryManager create recovery manger.
func newRecoveryManager(db dbm.DB, accountMgr *account.Manager) *recoveryManager {
	return &recoveryManager{
		db:         db,
		accountMgr: accountMgr,
		addresses:  make(map[bc.Hash]*account.CtrlProgram),
		state:      newRecoveryState(),
	}
}

func (m *recoveryManager) AddrResurrect(accts []*account.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, acct := range accts {
		m.state.stateForScope(acct)
		if err := m.extendScanAddresses(acct.ID, false); err != nil {
			return err
		}

		//Bip32 path no change field, no need to create addresses repeatedly.
		if acct.DeriveRule == signers.BIP0032 {
			continue
		}
		if err := m.extendScanAddresses(acct.ID, true); err != nil {
			return err
		}
	}

	m.state.StartTime = time.Now()
	if err := m.commitStatusInfo(); err != nil {
		return err
	}

	m.started = true
	return nil
}

func (m *recoveryManager) AcctResurrect(xPubs []chainkd.XPub) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.tryStartXPubsRec() {
		return ErrRecoveryBusy
	}

	m.state.XPubs = xPubs
	m.state.XPubsStatus = newBranchRecoveryState(acctRecoveryWindow)

	if err := m.extendScanAccounts(); err != nil {
		m.stopXPubsRec()
		return err
	}
	m.state.StartTime = time.Now()
	if err := m.commitStatusInfo(); err != nil {
		return err
	}

	m.started = true
	return nil
}

func (m *recoveryManager) commitStatusInfo() error {
	rawStatus, err := json.Marshal(m.state)
	if err != nil {
		return err
	}

	m.db.Set(recoveryKey, rawStatus)
	return nil
}

func genAcctAlias(xPubs []chainkd.XPub, index uint64) string {
	var tmp []byte
	for _, xPub := range xPubs {
		tmp = append(tmp, xPub[:6]...)
	}
	return fmt.Sprintf("%x:%x", tmp, index)
}

func (m *recoveryManager) extendScanAccounts() error {
	if m.state.XPubsStatus == nil {
		return nil
	}

	curHorizon, delta := m.state.XPubsStatus.extendHorizon()
	for index := curHorizon; index < curHorizon+delta; index++ {
		alias := genAcctAlias(m.state.XPubs, index)
		account, err := account.CreateAccount(m.state.XPubs, len(m.state.XPubs), alias, index, signers.BIP0044)
		if err != nil {
			return err
		}

		m.state.stateForScope(account)
		//generate resurrect address for new account.
		if err := m.extendScanAddresses(account.ID, true); err != nil {
			return err
		}

		if err := m.extendScanAddresses(account.ID, false); err != nil {
			return err
		}
	}

	return nil
}

func getCPHash(cp []byte) bc.Hash {
	var h [32]byte
	sha3pool.Sum256(h[:], cp)
	return bc.NewHash(h)
}

func (m *recoveryManager) extendAddress(acct *account.Account, index uint64, change bool) error {
	cp, err := account.CreateCtrlProgram(acct, index, change)
	if err != nil {
		return err
	}

	m.addresses[getCPHash(cp.ControlProgram)] = cp
	return nil
}

func (m *recoveryManager) extendScanAddresses(accountID string, change bool) error {
	state, ok := m.state.AccountsStatus[accountID]
	if !ok {
		return ErrInvalidAcctID
	}

	var curHorizon, delta uint64
	if change {
		curHorizon, delta = state.InternalBranch.extendHorizon()
	} else {
		curHorizon, delta = state.ExternalBranch.extendHorizon()
	}
	for index := curHorizon; index < curHorizon+delta; index++ {
		if err := m.extendAddress(state.Account, index, change); err != nil {
			return err
		}
	}
	return nil
}

func (m *recoveryManager) processBlock(b *types.Block) error {
	for _, tx := range b.Transactions {
		for _, output := range tx.Outputs {
			if cp, ok := m.addresses[getCPHash(output.ControlProgram)]; ok {
				status, ok := m.state.AccountsStatus[cp.AccountID]
				if !ok {
					return ErrInvalidAcctID
				}

				if err := m.reportFound(status.Account, cp); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// FilterRecoveryTxs Filter transactions that meet the recovery address
func (m *recoveryManager) FilterRecoveryTxs(b *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}
	if b.Time().After(m.state.StartTime) {
		m.finished()
		return nil
	}
	return m.processBlock(b)
}

func (m *recoveryManager) finished() {
	m.db.Delete(recoveryKey)
	m.started = false
	m.addresses = make(map[bc.Hash]*account.CtrlProgram)
	m.state = newRecoveryState()
	m.stopXPubsRec()
}

func (m *recoveryManager) LoadStatusInfo() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rawStatus := m.db.Get(recoveryKey)
	if rawStatus == nil {
		return nil
	}

	if err := json.Unmarshal(rawStatus, m.state); err != nil {
		return err
	}

	if m.state.XPubs != nil && !m.tryStartXPubsRec() {
		return ErrRecoveryBusy
	}

	if err := m.restoreAddresses(); err != nil {
		m.stopXPubsRec()
		return err
	}

	m.started = true
	return nil
}

// restoreAddresses resume addresses for unfinished tasks
func (m *recoveryManager) restoreAddresses() error {
	for _, state := range m.state.AccountsStatus {
		for index := uint64(1); index <= state.InternalBranch.Horizon; index++ {
			if err := m.extendAddress(state.Account, index, true); err != nil {
				return err
			}
		}

		for index := uint64(1); index <= state.ExternalBranch.Horizon; index++ {
			if err := m.extendAddress(state.Account, index, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// reportFound found your own address operation.
func (m *recoveryManager) reportFound(account *account.Account, cp *account.CtrlProgram) error {
	if m.state.XPubsStatus != nil && reflect.DeepEqual(m.state.XPubs, account.XPubs) {
		//recovery from XPubs need save account to db.
		if err := m.saveAccount(account); err != nil {
			return err
		}

		m.state.XPubsStatus.reportFound(account.KeyIndex)
		if err := m.extendScanAccounts(); err != nil {
			return err
		}
	}

	if cp.Change {
		m.state.AccountsStatus[account.ID].InternalBranch.reportFound(cp.KeyIndex)
	} else {
		m.state.AccountsStatus[account.ID].ExternalBranch.reportFound(cp.KeyIndex)
	}

	if err := m.extendScanAddresses(account.ID, cp.Change); err != nil {
		return err
	}

	if err := m.accountMgr.CreateBatchAddresses(account.ID, cp.Change, cp.KeyIndex); err != nil {
		return err
	}

	return m.commitStatusInfo()
}

func (m *recoveryManager) saveAccount(acct *account.Account) error {
	tmp, err := m.accountMgr.FindByID(acct.ID)
	if err != nil && errors.Root(err) != account.ErrFindAccount {
		return err
	}

	if tmp != nil {
		return nil
	}
	return m.accountMgr.SaveAccount(acct)
}

//tryStartXPubsRec guarantee that only one xPubs recovery is in progress.
func (m *recoveryManager) tryStartXPubsRec() bool {
	return atomic.CompareAndSwapInt32(&m.locked, 0, 1)
}

//stopXPubsRec release xPubs recovery lock.
func (m *recoveryManager) stopXPubsRec() {
	m.state.XPubs = nil
	atomic.StoreInt32(&m.locked, 0)
}
