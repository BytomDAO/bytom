package wallet

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/common"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
)

const (
	// defaultAcctRecoveryWindow defines the account derivation lookahead used when
	// attempting to recover the set of used accounts.
	defaultAcctRecoveryWindow = uint64(6)

	// defaultAddrRecoveryWindow defines the address derivation lookahead used when
	// attempting to recover the set of used addresses.
	defaultAddrRecoveryWindow = uint64(128)
)

var (
	//recoveryKey key for db store recovery info
	recoveryKey = []byte("RecoveryInfo")

	// KeyScopeBIP0032 is the key scope for BIP0032 derivation. BIP0032
	// compatible with old address generation methods.
	KeyScopeBIP0032 = KeyScope{
		Purpose: signers.BIP0032,
	}

	// KeyScopeBIP0044 is the key scope for BIP0044 derivation.
	KeyScopeBIP0044 = KeyScope{
		Purpose: signers.BIP0044,
	}

	// DefaultKeyScopes is the set of default key scopes that will be
	// created by the root manager upon initial creation.
	DefaultKeyScopes = []KeyScope{
		KeyScopeBIP0044,
	}
)

type addrPath struct {
	KeyScope
	acctIndex uint64
	change    bool
	addrIndex uint64
}

// KeyScope represents a restricted key scope from the primary root key within
// the HD chain. From the root manager (m/) we can create a nearly arbitrary
// number of ScopedKeyManagers of key derivation path: m/purpose'.
type KeyScope struct {
	// Purpose is the purpose of this key scope. This is the first child of
	// the master HD key.
	Purpose uint8
}

type AccountScope struct {
	AccountIndex uint64
	KeyScope
}

type scope map[AccountScope]*ScopeRecoveryState

func (s *scope) MarshalText() ([]byte, error) {
	scopes := make(map[string]*ScopeRecoveryState, len(*s))
	for k, v := range *s {
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}

		scopes[string(key)] = v
	}
	return json.Marshal(scopes)
}

func (s *scope) UnmarshalText(text []byte) error {
	var scopes map[string]*ScopeRecoveryState
	if err := json.Unmarshal(text, &scopes); err != nil {
		return err
	}

	for k, v := range scopes {
		acctScope := &AccountScope{}
		if err := json.Unmarshal([]byte(k), acctScope); err != nil {
			return err
		}

		(*s)[*acctScope] = v
	}
	return nil
}

type RecoveryManager struct {
	RWMutex sync.RWMutex

	db db.DB

	// recoveryWindow defines the key-derivation lookahead used when
	// attempting to recover the set of used addresses.
	recoveryWindow uint64

	// started is true after the first block has been added to the batch.
	started bool

	// state encapsulates and allocates the necessary recovery state for all
	// key scopes and subsidiary derivation paths.
	state *RecoveryState

	//addresses all addresses derivation lookahead used when
	// attempting to recover the set of used addresses.
	addresses map[common.Hash]addrPath
}

func NewRecoveryManager(db db.DB) *RecoveryManager {
	return &RecoveryManager{
		db:             db,
		recoveryWindow: defaultAddrRecoveryWindow,
		addresses:      make(map[common.Hash]addrPath),
	}
}

func (rm *RecoveryManager) checkAddress(hash common.Hash) (addrPath, bool) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	path, ok := rm.addresses[hash]
	return path, ok
}

func (rm *RecoveryManager) checkAccount(acctIndex uint64) (string, bool) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	acctID, ok := rm.state.Accounts.FoundAccounts[acctIndex]
	return acctID, ok
}

func (rm *RecoveryManager) commitStatusInfo(storeBatch db.Batch) error {
	rawStatus, err := json.Marshal(rm.state)
	if err != nil {
		return err
	}

	if storeBatch != nil {
		storeBatch.Set(recoveryKey, rawStatus)
		return nil
	}

	rm.db.Set(recoveryKey, rawStatus)
	return nil
}

func (rm *RecoveryManager) extendScanAccounts() {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	curHorizon, delta := rm.state.Accounts.ExtendHorizon()
	for index := curHorizon; index < curHorizon+delta; index++ {
		for _, keyScope := range DefaultKeyScopes {
			accountScope := AccountScope{AccountIndex: index, KeyScope: keyScope}
			rm.state.StateForScope(accountScope)
		}
	}
}

func (rm *RecoveryManager) extendScanAddresses(restore bool) error {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	for scope, state := range rm.state.Scopes {
		var start, stop uint64

		if restore {
			start = uint64(0)
			stop = state.InternalBranch.Horizon + 1
		} else {
			curHorizon, delta := state.InternalBranch.ExtendHorizon()
			start = curHorizon
			stop = curHorizon + delta
		}
		for index := start; index < stop; index++ {
			cp, err := account.CreateRecoveryAddress([]chainkd.XPub{rm.state.XPub}, scope.Purpose, scope.AccountIndex, true, index)
			if err != nil {
				return err
			}

			var hash common.Hash
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			rm.addresses[hash] = addrPath{acctIndex: scope.AccountIndex, change: true, addrIndex: cp.KeyIndex, KeyScope: scope.KeyScope}
		}

		if restore {
			start = uint64(0)
			stop = state.ExternalBranch.Horizon + 1
		} else {
			curHorizon, delta := state.ExternalBranch.ExtendHorizon()
			start = curHorizon
			stop = curHorizon + delta
		}
		for index := start; index < stop; index++ {
			cp, err := account.CreateRecoveryAddress([]chainkd.XPub{rm.state.XPub}, scope.Purpose, scope.AccountIndex, false, index)
			if err != nil {
				return err
			}

			var hash common.Hash
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			rm.addresses[hash] = addrPath{acctIndex: scope.AccountIndex, change: false, addrIndex: cp.KeyIndex, KeyScope: scope.KeyScope}
		}
	}
	return nil
}

func (rm *RecoveryManager) isFinished() bool {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	return rm.state.Finished
}

func (rm *RecoveryManager) IsStarted() bool {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	return rm.started
}

func (rm *RecoveryManager) loadStatusInfo() (bool, error) {
	rawStatus := rm.db.Get(recoveryKey)
	if rawStatus == nil {
		return false, nil
	}

	status := NewRecoveryState()
	if err := json.Unmarshal(rawStatus, status); err != nil {
		return false, err
	}

	rm.state = status
	return true, nil
}

func (rm *RecoveryManager) resurrectFinished() error {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.state.Finished = true
	if err := rm.commitStatusInfo(nil); err != nil {
		return err
	}

	rm.started = false
	return nil
}

func (rm *RecoveryManager) resurrectStart() {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.started = true
}

// Resurrect restores all known addresses for the provided scopes that can be
// found in the walletdb namespace. This method ensures that the recovery state's
// horizons properly start from the last found address of a prior recovery
// attempt.
func (rm *RecoveryManager) Resurrect() error {
	rm.extendScanAccounts()
	if err := rm.extendScanAddresses(false); err != nil {
		return err
	}

	rm.resurrectStart()
	return nil
}

// ReportFound updates the last found index if the reported index exceeds the
// current value.
func (rm *RecoveryManager) ReportFound(acctIndex uint64, change bool, addrIndex uint64) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	acctScope := AccountScope{
		AccountIndex: acctIndex,
		KeyScope:     KeyScopeBIP0044,
	}
	rm.state.Accounts.ReportFound(acctIndex)
	if change {
		rm.state.Scopes[acctScope].InternalBranch.ReportFound(addrIndex)
	} else {
		rm.state.Scopes[acctScope].ExternalBranch.ReportFound(addrIndex)
	}
}

func (rm *RecoveryManager) setAccount(acctIndex uint64, acctID string) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.state.Accounts.FoundAccounts[acctIndex] = acctID
}

func (rm *RecoveryManager) StatusInit(XPub chainkd.XPub) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.state = NewRecoveryState()
	rm.state.XPub = XPub
}

func (rm *RecoveryManager) startTime() time.Time {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	return rm.state.StartTime
}

type RecoveryState struct {
	Finished bool

	// The time to start the recovery task, used to determine whether
	// recovery task is completed.
	StartTime time.Time

	//Root XPub for restoring the wallet
	XPub chainkd.XPub

	// AddrRecoveryWindow defines the key-derivation lookahead used when
	// attempting to recover the set of used addresses. This value will be
	// used to instantiate a new RecoveryState for each requested scope.
	AddrRecoveryWindow uint64

	Accounts *AccountRecoveryState

	// scopes maintains a map of each requested key scope to its active
	// RecoveryState.
	Scopes scope
}

func NewRecoveryState() *RecoveryState {
	return &RecoveryState{
		AddrRecoveryWindow: defaultAddrRecoveryWindow,
		Accounts:           NewAccountRecoveryState(defaultAcctRecoveryWindow),
		Scopes:             make(map[AccountScope]*ScopeRecoveryState),
		StartTime:          time.Now(),
	}
}

// StateForScope returns a ScopeRecoveryState for the provided key scope. If one
// does not already exist, a new one will be generated with the RecoveryState's
// recoveryWindow.
func (rs *RecoveryState) StateForScope(
	accountScope AccountScope) *ScopeRecoveryState {

	// If the account recovery state already exists, return it.
	if scopeState, ok := rs.Scopes[accountScope]; ok {
		return scopeState
	}

	// Otherwise, initialize the recovery state for this scope with the
	// chosen recovery window.
	rs.Scopes[accountScope] = NewScopeRecoveryState(rs.AddrRecoveryWindow)

	return rs.Scopes[accountScope]
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
}

func NewScopeRecoveryState(recoveryWindow uint64) *ScopeRecoveryState {
	return &ScopeRecoveryState{
		ExternalBranch: NewBranchRecoveryState(recoveryWindow),
		InternalBranch: NewBranchRecoveryState(recoveryWindow),
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

type AccountRecoveryState struct {
	BranchRecoveryState

	// addresses is a map of child index to address for all actively watched
	// addresses belonging to this branch.
	FoundAccounts map[uint64]string
}

// NewBranchRecoveryState creates a new BranchRecoveryState that can be used to
// track either the external or internal branch of an account's derivation path.
func NewAccountRecoveryState(recoveryWindow uint64) *AccountRecoveryState {
	return &AccountRecoveryState{
		BranchRecoveryState: BranchRecoveryState{RecoveryWindow: recoveryWindow},
		FoundAccounts:       make(map[uint64]string),
	}
}
