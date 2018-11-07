package wallet

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/common"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol/bc/types"
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
	//recoveryKey key for db store recovery info.
	recoveryKey = []byte("RecoveryInfo")

	// DefaultDeriveRules is the derived rule for getting the
	// child key.
	DefaultDeriveRules = []uint8{
		signers.BIP0032,
		signers.BIP0044,
	}
)

type addrPath struct {
	xPubs      []chainkd.XPub
	deriveRule uint8
	acctIndex  uint64
	change     bool
	addrIndex  uint64
}

// AccountScope Is used to describe an account in a hierarchical deterministic wallets.
type AccountScope struct {
	XPubsHash    string
	DeriveRule   uint8
	AccountIndex uint64
}

func xPubsHash(xPubs []chainkd.XPub) string {
	cpy := append([]chainkd.XPub{}, xPubs[:]...)
	sort.Sort(signers.SortKeys(cpy))
	var slice []byte
	for _, xPub := range xPubs {
		slice = append(slice, xPub[:]...)
	}
	var hash common.Hash
	sha3pool.Sum256(hash[:], slice)
	return hash.Str()
}

type accountStatus map[AccountScope]*ScopeRecoveryState

func (as *accountStatus) MarshalText() ([]byte, error) {
	scopes := make(map[string]*ScopeRecoveryState, len(*as))
	for k, v := range *as {
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}

		scopes[string(key)] = v
	}
	return json.Marshal(scopes)
}

func (as *accountStatus) UnmarshalText(text []byte) error {
	var scopes map[string]*ScopeRecoveryState
	if err := json.Unmarshal(text, &scopes); err != nil {
		return err
	}

	for k, v := range scopes {
		acctScope := &AccountScope{}
		if err := json.Unmarshal([]byte(k), acctScope); err != nil {
			return err
		}

		(*as)[*acctScope] = v
	}
	return nil
}

// RecoveryManager manage recovery wallet from key.
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

// newRecoveryManager create recovery manger.
func newRecoveryManager(db db.DB) *RecoveryManager {
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

func (rm *RecoveryManager) checkAccount(xPubs []chainkd.XPub, acctIndex uint64, accountMgr *account.Manager) (*string, error) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	status, ok := rm.state.XPubsStatus[xPubsHash(xPubs)]
	if ok {
		acctID, ok := status.FoundAccounts[acctIndex]
		if ok {
			return &acctID, nil
		}
	}

	accounts, err := accountMgr.GetAccountByXPubs(xPubs)
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		if account.KeyIndex == acctIndex {
			return &account.ID, nil
		}
	}

	return nil, nil
}

func (rm *RecoveryManager) commitStatusInfo() error {
	rawStatus, err := json.Marshal(rm.state)
	if err != nil {
		return err
	}

	rm.db.Set(recoveryKey, rawStatus)
	return nil
}

func (rm *RecoveryManager) extendScanAccounts() {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	for _, state := range rm.state.XPubsStatus {
		curHorizon, delta := state.ExtendHorizon()
		for index := curHorizon; index < curHorizon+delta; index++ {
			for _, deriveRule := range DefaultDeriveRules {
				accountScope := AccountScope{XPubsHash: xPubsHash(state.XPubs), AccountIndex: index, DeriveRule: deriveRule}
				rm.state.StateForScope(accountScope, state.XPubs)
			}
		}
	}
}

func (rm *RecoveryManager) extendScanAddresses() error {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	for scope, state := range rm.state.AccountsStatus {
		curHorizon, delta := state.InternalBranch.ExtendHorizon()
		for index := curHorizon; index < curHorizon+delta; index++ {
			cp, err := account.CreateRecoveryAddress(state.XPubs, scope.DeriveRule, scope.AccountIndex, true, index)
			if err != nil {
				return err
			}

			var hash common.Hash
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			rm.addresses[hash] = addrPath{xPubs: state.XPubs, acctIndex: scope.AccountIndex, change: true, addrIndex: cp.KeyIndex, deriveRule: scope.DeriveRule}
		}

		curHorizon, delta = state.ExternalBranch.ExtendHorizon()
		for index := curHorizon; index < curHorizon+delta; index++ {
			cp, err := account.CreateRecoveryAddress(state.XPubs, scope.DeriveRule, scope.AccountIndex, false, index)
			if err != nil {
				return err
			}

			var hash common.Hash
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			rm.addresses[hash] = addrPath{xPubs: state.XPubs, acctIndex: scope.AccountIndex, change: false, addrIndex: cp.KeyIndex, deriveRule: scope.DeriveRule}
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
			var hash [32]byte
			sha3pool.Sum256(hash[:], output.ControlProgram)
			if path, ok := rm.checkAddress(hash); ok {
				var accountID string
				acctID, err := rm.checkAccount(path.xPubs, path.acctIndex, accountMgr)
				if err != nil {
					return err
				}

				if acctID == nil {
					var tmp []byte
					for _, xPub := range path.xPubs {
						tmp = append(tmp, xPub[:6]...)
					}
					alias := fmt.Sprintf("%x:%x", tmp, path.acctIndex)
					account, err := accountMgr.Create(path.xPubs, len(path.xPubs), alias, path.acctIndex, path.deriveRule)
					if err != nil {
						return err
					}
					accountID = account.ID
				} else {
					accountID = *acctID
				}

				rm.setAccount(path.xPubs, path.acctIndex, accountID)
				rm.ReportFound(path.xPubs, path.deriveRule, path.acctIndex, path.change, path.addrIndex)
				rm.extendScanAccounts()
				if err := rm.extendScanAddresses(); err != nil {
					return err
				}

				if err := rm.commitStatusInfo(); err != nil {
					return err
				}

				if err := accountMgr.CreateBatchAddresses(accountID, path.change, path.addrIndex); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (rm *RecoveryManager) isFinished() bool {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	return rm.state.Finished
}

// IsStarted used to determine if recovery is in progress.
func (rm *RecoveryManager) IsStarted() bool {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	return rm.started
}

func (rm *RecoveryManager) loadStatusInfo() error {
	rawStatus := rm.db.Get(recoveryKey)
	if rawStatus == nil {
		return nil
	}

	status := newRecoveryState()
	if err := json.Unmarshal(rawStatus, status); err != nil {
		return err
	}

	rm.state = status
	if !rm.isFinished() {
		if err := rm.restoreAddresses(); err != nil {
			return err
		}

		rm.resurrectStart()
	}

	return nil
}

func (rm *RecoveryManager) restoreAddresses() error {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	for scope, state := range rm.state.AccountsStatus {
		for index := uint64(0); index <= state.InternalBranch.Horizon; index++ {
			cp, err := account.CreateRecoveryAddress(state.XPubs, scope.DeriveRule, scope.AccountIndex, true, index)
			if err != nil {
				return err
			}

			var hash common.Hash
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			rm.addresses[hash] = addrPath{xPubs: state.XPubs, acctIndex: scope.AccountIndex, change: true, addrIndex: cp.KeyIndex, deriveRule: scope.DeriveRule}
		}

		for index := uint64(0); index <= state.ExternalBranch.Horizon; index++ {
			cp, err := account.CreateRecoveryAddress(state.XPubs, scope.DeriveRule, scope.AccountIndex, false, index)
			if err != nil {
				return err
			}

			var hash common.Hash
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			rm.addresses[hash] = addrPath{xPubs: state.XPubs, acctIndex: scope.AccountIndex, change: false, addrIndex: cp.KeyIndex, deriveRule: scope.DeriveRule}
		}
	}
	return nil
}

func (rm *RecoveryManager) resurrectFinished() error {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.state.Finished = true
	if err := rm.commitStatusInfo(); err != nil {
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
	if err := rm.extendScanAddresses(); err != nil {
		return err
	}

	rm.resurrectStart()
	return nil
}

// ReportFound updates the last found index if the reported index exceeds the
// current value.
func (rm *RecoveryManager) ReportFound(xPubs []chainkd.XPub, deriveRule uint8, acctIndex uint64, change bool, addrIndex uint64) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()
	key := xPubsHash(xPubs)
	acctScope := AccountScope{
		XPubsHash:    key,
		AccountIndex: acctIndex,
		DeriveRule:   deriveRule,
	}
	if _, ok := rm.state.XPubsStatus[key]; ok {
		rm.state.XPubsStatus[key].ReportFound(acctIndex)
	}
	if change {
		rm.state.AccountsStatus[acctScope].InternalBranch.ReportFound(addrIndex)
	} else {
		rm.state.AccountsStatus[acctScope].ExternalBranch.ReportFound(addrIndex)
	}
}

func (rm *RecoveryManager) setAccount(xPubs []chainkd.XPub, acctIndex uint64, acctID string) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	if _, ok := rm.state.XPubsStatus[xPubsHash(xPubs)]; !ok {
		return
	}
	rm.state.XPubsStatus[xPubsHash(xPubs)].FoundAccounts[acctIndex] = acctID
}

// StatusInit init recovery status manager.
func (rm *RecoveryManager) StatusInit(xPubs []chainkd.XPub) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.state = newRecoveryState()
	rm.state.XPubsStatus[xPubsHash(xPubs)] = newAccountRecoveryState(defaultAcctRecoveryWindow, xPubs)
}

// AcctStatusInit init recovery status for account address rescan.
func (rm *RecoveryManager) AcctStatusInit(accts []*account.Account) {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	rm.state = newRecoveryState()
	for _, acct := range accts {
		accountScope := AccountScope{XPubsHash: xPubsHash(acct.XPubs), AccountIndex: acct.KeyIndex, DeriveRule: acct.DeriveRule}
		rm.state.StateForScope(accountScope, acct.XPubs)
	}
}

func (rm *RecoveryManager) startTime() time.Time {
	rm.RWMutex.Lock()
	defer rm.RWMutex.Unlock()

	return rm.state.StartTime
}

// RecoveryState used to record the status of a recovery process.
type RecoveryState struct {
	Finished bool

	// The time to start the recovery task, used to determine whether
	// recovery task is completed.
	StartTime time.Time

	// AddrRecoveryWindow defines the key-derivation lookahead used when
	// attempting to recover the set of used addresses. This value will be
	// used to instantiate a new RecoveryState for each requested scope.
	AddrRecoveryWindow uint64

	// XPubsStatus maintains a map of each requested XPub to its active
	// account recovery state.
	XPubsStatus map[string]*AccountRecoveryState

	// AcctStatus maintains a map of each requested key scope to its active
	// recovery state.
	AccountsStatus accountStatus
}

func newRecoveryState() *RecoveryState {
	return &RecoveryState{
		AddrRecoveryWindow: defaultAddrRecoveryWindow,
		XPubsStatus:        make(map[string]*AccountRecoveryState),
		AccountsStatus:     make(map[AccountScope]*ScopeRecoveryState),
		StartTime:          time.Now(),
	}
}

// StateForScope returns a ScopeRecoveryState for the provided key scope. If one
// does not already exist, a new one will be generated with the RecoveryState's
// recoveryWindow.
func (rs *RecoveryState) StateForScope(accountScope AccountScope, xPubs []chainkd.XPub) *ScopeRecoveryState {
	// If the account recovery state already exists, return it.
	if scopeState, ok := rs.AccountsStatus[accountScope]; ok {
		return scopeState
	}

	// Otherwise, initialize the recovery state for this scope with the
	// chosen recovery window.
	rs.AccountsStatus[accountScope] = newScopeRecoveryState(rs.AddrRecoveryWindow, xPubs)

	return rs.AccountsStatus[accountScope]
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

	// XPubs account signer xPubs
	XPubs []chainkd.XPub
}

func newScopeRecoveryState(recoveryWindow uint64, xPubs []chainkd.XPub) *ScopeRecoveryState {
	return &ScopeRecoveryState{
		ExternalBranch: NewBranchRecoveryState(recoveryWindow),
		InternalBranch: NewBranchRecoveryState(recoveryWindow),
		XPubs:          xPubs,
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

// AccountRecoveryState for recording key of the hd wallet account recovery status.
type AccountRecoveryState struct {
	BranchRecoveryState

	// addresses is a map of child index to address for all actively watched
	// addresses belonging to this branch.
	FoundAccounts map[uint64]string

	// XPubs account signer xPubs
	XPubs []chainkd.XPub
}

// newAccountRecoveryState creates a new AccountRecoveryState that can be used to
// track account recovery status.
func newAccountRecoveryState(recoveryWindow uint64, xPubs []chainkd.XPub) *AccountRecoveryState {
	return &AccountRecoveryState{
		BranchRecoveryState: BranchRecoveryState{RecoveryWindow: recoveryWindow},
		FoundAccounts:       make(map[uint64]string),
		XPubs:               xPubs,
	}
}
