package wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	// defaultAcctRecoveryWindow defines the account derivation lookahead used when
	// attempting to recover the set of used accounts.
	defaultAcctRecoveryWindow = uint64(6)

	// defaultAddrRecoveryWindow defines the address derivation lookahead used when
	// attempting to recover the set of used addresses.
	defaultAddrRecoveryWindow = uint64(128)

	LockedFlag   = int32(1)
	UnlockedFlag = int32(0)
)

//recoveryKey key for db store recovery info.
var recoveryKey = []byte("RecoveryInfo")

// RecoveryManager manage recovery wallet from key.
type RecoveryManager struct {
	mu *Mutex

	db db.DB

	// recoveryWindow defines the key-derivation lookahead used when
	// attempting to recover the set of used addresses.
	recoveryWindow uint64

	started bool

	// state encapsulates and allocates the necessary recovery state for all
	// key scopes and subsidiary derivation paths.
	state *RecoveryState

	//addresses all addresses derivation lookahead used when
	// attempting to recover the set of used addresses.
	addresses map[bc.Hash]*account.CtrlProgram
}

type Mutex struct {
	in     sync.Mutex
	status *int32
}

func NewMutex() *Mutex {
	status := UnlockedFlag
	return &Mutex{
		status: &status,
	}
}

func (m *Mutex) Unlock() {
	m.in.Unlock()
	atomic.AddInt32(m.status, UnlockedFlag)
}

func (m *Mutex) TryLock() bool {
	if atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.in)), UnlockedFlag, LockedFlag) {
		atomic.AddInt32(m.status, LockedFlag)
		return true
	}
	return false
}

// AcctStatusInit init recovery status for account address rescan.
func (rm *RecoveryManager) AcctStatusInit(accts []*account.Account) {
	rm.state = newRecoveryState()
	for _, acct := range accts {
		rm.state.StateForScope(acct)
	}
}

// newRecoveryManager create recovery manger.
func newRecoveryManager(db db.DB) *RecoveryManager {
	return &RecoveryManager{
		mu:             NewMutex(),
		db:             db,
		recoveryWindow: defaultAddrRecoveryWindow,
		addresses:      make(map[bc.Hash]*account.CtrlProgram),
	}
}

func (rm *RecoveryManager) checkAddress(hash bc.Hash) (*account.CtrlProgram, bool) {
	cp, ok := rm.addresses[hash]
	return cp, ok
}

func (rm *RecoveryManager) saveAccount(accountMgr *account.Manager, accountID string) (*account.Account, error) {
	account, err := accountMgr.FindByID(accountID)
	if err == nil && account != nil {
		return account, nil
	}

	if state, ok := rm.state.AccountsStatus[accountID]; ok {
		if err := accountMgr.SaveAccount(state.Account); err != nil {
			return nil, err
		}

		return state.Account, nil
	}
	return nil, errors.New("save account err")
}

func (rm *RecoveryManager) commitStatusInfo() error {
	rawStatus, err := json.Marshal(rm.state)
	if err != nil {
		return err
	}

	rm.db.Set(recoveryKey, rawStatus)
	return nil
}

func genAcctAlias(xPubs []chainkd.XPub, index uint64) string {
	var tmp []byte
	for _, xPub := range xPubs {
		tmp = append(tmp, xPub[:6]...)
	}
	return fmt.Sprintf("%x:%x", tmp, index)
}

func (rm *RecoveryManager) extendScanAccounts(accountMgr *account.Manager) error {
	if rm.state.XPubsStatus == nil {
		return nil
	}

	curHorizon, delta := rm.state.XPubsStatus.AccountBranch.ExtendHorizon()
	for index := curHorizon; index < curHorizon+delta; index++ {
		alias := genAcctAlias(rm.state.XPubs, index)
		account, err := accountMgr.CreateAccount(rm.state.XPubs, len(rm.state.XPubs), alias, index, signers.BIP0044)
		if err != nil {
			return err
		}

		rm.state.StateForScope(account)
	}
	return nil
}

func getCPHash(cp []byte) bc.Hash {
	var h [32]byte
	sha3pool.Sum256(h[:], cp)
	return bc.NewHash(h)
}

func (rm *RecoveryManager) extendAddresses(accountMgr *account.Manager, account *account.Account, index uint64, change bool) error {
	cp, err := accountMgr.CreateCtrlProgram(account, index, change)
	if err != nil {
		return err
	}

	rm.addresses[getCPHash(cp.ControlProgram)] = cp
	return nil
}

func (rm *RecoveryManager) extendScanAddresses(accountMgr *account.Manager) error {
	for _, state := range rm.state.AccountsStatus {
		curHorizon, delta := state.InternalBranch.ExtendHorizon()
		for index := curHorizon; index < curHorizon+delta; index++ {
			if err := rm.extendAddresses(accountMgr, state.Account, index, true); err != nil {
				return err
			}
		}

		curHorizon, delta = state.ExternalBranch.ExtendHorizon()
		for index := curHorizon; index < curHorizon+delta; index++ {
			if err := rm.extendAddresses(accountMgr, state.Account, index, false); err != nil {
				return err
			}
		}
	}

	return nil
}

// filterRecoveryTxs Filter transactions that meet the recovery address
func (rm *RecoveryManager) filterRecoveryTxs(b *types.Block, accountMgr *account.Manager) error {
	if !rm.IsStarted() {
		return nil
	}

	if b.Time().After(rm.startTime()) {
		return rm.resurrectFinished()
	}

	for _, tx := range b.Transactions {
		for _, output := range tx.Outputs {
			if cp, ok := rm.checkAddress(getCPHash(output.ControlProgram)); ok {
				account, err := rm.saveAccount(accountMgr, cp.AccountID)
				if err != nil {
					return err
				}

				rm.ReportFound(account, cp)
				if err := rm.extendScanAccounts(accountMgr); err != nil {
					return err
				}

				if err := rm.extendScanAddresses(accountMgr); err != nil {
					return err
				}

				if err := rm.commitStatusInfo(); err != nil {
					return err
				}

				if err := accountMgr.CreateBatchAddresses(account.ID, cp.Change, cp.KeyIndex); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (rm *RecoveryManager) isFinished() bool {
	return rm.state.Finished
}

// IsStarted used to determine if recovery is in progress.
func (rm *RecoveryManager) IsStarted() bool {
	return rm.started
}

func (rm *RecoveryManager) loadStatusInfo(accountMgr *account.Manager) error {
	if !rm.TryLock() {
		return nil
	}
	rawStatus := rm.db.Get(recoveryKey)
	if rawStatus == nil {
		rm.UnLock()
		return nil
	}

	status := newRecoveryState()
	if err := json.Unmarshal(rawStatus, status); err != nil {
		return err
	}

	rm.state = status
	if !rm.isFinished() {
		if err := rm.restoreAddresses(accountMgr); err != nil {
			return err
		}

		rm.resurrectStart()
	}

	return nil
}

func (rm *RecoveryManager) restoreAddresses(accountMgr *account.Manager) error {
	for _, state := range rm.state.AccountsStatus {
		for index := uint64(0); index <= state.InternalBranch.Horizon; index++ {
			if err := rm.extendAddresses(accountMgr, state.Account, index, true); err != nil {
				return err
			}
		}

		for index := uint64(0); index <= state.ExternalBranch.Horizon; index++ {
			if err := rm.extendAddresses(accountMgr, state.Account, index, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rm *RecoveryManager) resurrectFinished() error {
	rm.state.Finished = true
	if err := rm.commitStatusInfo(); err != nil {
		return err
	}

	rm.started = false
	rm.UnLock()
	return nil
}

func (rm *RecoveryManager) resurrectStart() {
	rm.started = true
}

// Resurrect restores all known addresses for the provided scopes that can be
// found in the walletdb namespace. This method ensures that the recovery state's
// horizons properly start from the last found address of a prior recovery
// attempt.
func (rm *RecoveryManager) Resurrect(accountMgr *account.Manager) error {
	rm.extendScanAccounts(accountMgr)
	if err := rm.extendScanAddresses(accountMgr); err != nil {
		rm.UnLock()
		return err
	}

	rm.resurrectStart()
	return nil
}

// ReportFound updates the last found index if the reported index exceeds the
// current value.
func (rm *RecoveryManager) ReportFound(account *account.Account, cp *account.CtrlProgram) {
	if rm.state.XPubsStatus == nil {
		return
	}

	rm.state.XPubsStatus.AccountBranch.ReportFound(account.KeyIndex)
	if cp.Change {
		rm.state.AccountsStatus[account.ID].InternalBranch.ReportFound(cp.KeyIndex)
	} else {
		rm.state.AccountsStatus[account.ID].ExternalBranch.ReportFound(cp.KeyIndex)
	}
}

// StatusInit init recovery status manager.
func (rm *RecoveryManager) StatusInit(xPubs []chainkd.XPub) {
	rm.state = newRecoveryState()
	rm.state.XPubs = xPubs
	rm.state.XPubsStatus = newAccountRecoveryState(defaultAcctRecoveryWindow)
}

func (rm *RecoveryManager) startTime() time.Time {
	return rm.state.StartTime
}

//TryLock try to lock recovery mgr, will not block
func (rm *RecoveryManager) TryLock() bool {
	//return rm.mu.TryLock()
	return true
}

func (rm *RecoveryManager) UnLock() {
	//rm.mu.Unlock()
}

// RecoveryState used to record the status of a recovery process.
type RecoveryState struct {
	Finished bool

	// XPubs recovery account xPubs
	XPubs []chainkd.XPub

	// The time to start the recovery task, used to determine whether
	// recovery task is completed.
	StartTime time.Time

	// XPubsStatus maintains a map of each requested XPub to its active
	// account recovery state.
	XPubsStatus *AccountRecoveryState

	// AcctStatus maintains a map of each requested key scope to its active
	// recovery state.
	AccountsStatus map[string]*ScopeRecoveryState
}

func newRecoveryState() *RecoveryState {
	return &RecoveryState{
		AccountsStatus: make(map[string]*ScopeRecoveryState),
		StartTime:      time.Now(),
	}
}

// StateForScope returns a ScopeRecoveryState for the provided key scope. If one
// does not already exist, a new one will be generated with the RecoveryState's
// recoveryWindow.
func (rs *RecoveryState) StateForScope(account *account.Account) *ScopeRecoveryState {
	// If the account recovery state already exists, return it.
	if scopeState, ok := rs.AccountsStatus[account.ID]; ok {
		return scopeState
	}

	// Otherwise, initialize the recovery state for this scope with the
	// chosen recovery window.
	rs.AccountsStatus[account.ID] = newScopeRecoveryState(defaultAddrRecoveryWindow, account)

	return rs.AccountsStatus[account.ID]
}

// ScopeRecoveryState is used to manage the recovery of addresses generated
// under a particular BIP32/BIP44 account. Each account tracks both an external and
// internal branch recovery state, both of which use the same recovery window.
type ScopeRecoveryState struct {
	// ExternalBranch is the recovery state of addresses generated for
	// external use, i.e. receiving addresses.
	ExternalBranch *BranchRecoveryState

	// InternalBranch is the recovery state of addresses generated for
	// internal use, i.e. change addresses.
	InternalBranch *BranchRecoveryState

	Account *account.Account
}

func newScopeRecoveryState(recoveryWindow uint64, account *account.Account) *ScopeRecoveryState {
	return &ScopeRecoveryState{
		ExternalBranch: NewBranchRecoveryState(recoveryWindow),
		InternalBranch: NewBranchRecoveryState(recoveryWindow),
		Account:        account,
	}
}

// BranchRecoveryState maintains the required state in-order to properly
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
type BranchRecoveryState struct {
	// recoveryWindow defines the key-derivation lookahead used when
	// attempting to recover the set of addresses on this branch.
	RecoveryWindow uint64

	// horizion records the highest child index watched by this branch.
	Horizon uint64

	// nextUnfound maintains the child index of the successor to the highest
	// index that has been found during recovery of this branch.
	NextUnfound uint64
}

// NewBranchRecoveryState creates a new BranchRecoveryState that can be used to
// track either the external or internal branch of an account's derivation path.
func NewBranchRecoveryState(recoveryWindow uint64) *BranchRecoveryState {
	return &BranchRecoveryState{
		RecoveryWindow: recoveryWindow,
		Horizon:        1,
		NextUnfound:    1,
	}
}

// ExtendHorizon returns the current horizon and the number of addresses that
// must be derived in order to maintain the desired recovery window.
func (brs *BranchRecoveryState) ExtendHorizon() (uint64, uint64) {

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

// ReportFound updates the last found index if the reported index exceeds the
// current value.
func (brs *BranchRecoveryState) ReportFound(index uint64) {
	if index >= brs.NextUnfound {
		brs.NextUnfound = index + 1
	}
}

// AccountRecoveryState for recording key of the hd wallet account recovery status.
type AccountRecoveryState struct {
	AccountBranch *BranchRecoveryState

	//// addresses is a map of child index to address for all actively watched
	//// addresses belonging to this branch.
	//FoundAccounts map[uint64]string
}

// newAccountRecoveryState creates a new AccountRecoveryState that can be used to
// track account recovery status.
func newAccountRecoveryState(recoveryWindow uint64) *AccountRecoveryState {
	return &AccountRecoveryState{
		AccountBranch: NewBranchRecoveryState(recoveryWindow),
	}
}
