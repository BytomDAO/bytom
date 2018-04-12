package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-wire/data/base58"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

//SINGLE single sign
const SINGLE = 1

//RecoveryIndex walletdb recovery cp number
const RecoveryIndex = 5000

var walletKey = []byte("walletInfo")
var privKeyKey = []byte("keysInfo")

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	WorkHeight uint64
	WorkHash   bc.Hash
	BestHeight uint64
	BestHash   bc.Hash
}

//KeyInfo is key import status
type KeyInfo struct {
	Alias    string       `json:"alias"`
	XPub     chainkd.XPub `json:"xpub"`
	Percent  uint8        `json:"percent"`
	Complete bool         `json:"complete"`
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB             db.DB
	status         StatusInfo
	AccountMgr     *account.Manager
	AssetReg       *asset.Registry
	Hsm            *pseudohsm.HSM
	chain          *protocol.Chain
	rescanProgress chan struct{}
	ImportPrivKey  bool
	keysInfo       []KeyInfo
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, account *account.Manager, asset *asset.Registry, hsm *pseudohsm.HSM, chain *protocol.Chain) (*Wallet, error) {
	w := &Wallet{
		DB:             walletDB,
		AccountMgr:     account,
		AssetReg:       asset,
		chain:          chain,
		Hsm:            hsm,
		rescanProgress: make(chan struct{}, 1),
		keysInfo:       make([]KeyInfo, 0),
	}

	if err := w.loadWalletInfo(); err != nil {
		return nil, err
	}

	if err := w.loadKeysInfo(); err != nil {
		return nil, err
	}

	w.ImportPrivKey = w.getImportKeyFlag()

	go w.walletUpdater()

	return w, nil
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) loadWalletInfo() error {
	if rawWallet := w.DB.Get(walletKey); rawWallet != nil {
		return json.Unmarshal(rawWallet, &w.status)
	}

	block, err := w.chain.GetBlockByHeight(0)
	if err != nil {
		return err
	}
	return w.AttachBlock(block)
}

func (w *Wallet) commitWalletInfo(batch db.Batch) error {
	rawWallet, err := json.Marshal(w.status)
	if err != nil {
		log.WithField("err", err).Error("save wallet info")
		return err
	}

	batch.Set(walletKey, rawWallet)
	batch.Write()
	return nil
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) loadKeysInfo() error {
	if rawKeyInfo := w.DB.Get(privKeyKey); rawKeyInfo != nil {
		json.Unmarshal(rawKeyInfo, &w.keysInfo)
		return nil
	}
	return nil
}

func (w *Wallet) commitkeysInfo() error {
	rawKeysInfo, err := json.Marshal(w.keysInfo)
	if err != nil {
		log.WithField("err", err).Error("save wallet info")
		return err
	}
	w.DB.Set(privKeyKey, rawKeysInfo)
	return nil
}

func (w *Wallet) getImportKeyFlag() bool {
	for _, v := range w.keysInfo {
		if v.Complete == false {
			return true
		}
	}
	return false
}

// AttachBlock attach a new block
func (w *Wallet) AttachBlock(block *types.Block) error {
	if block.PreviousBlockHash != w.status.WorkHash {
		log.Warn("wallet skip attachBlock due to status hash not equal to previous hash")
		return nil
	}

	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	storeBatch := w.DB.NewBatch()
	w.indexTransactions(storeBatch, block, txStatus)
	w.buildAccountUTXOs(storeBatch, block, txStatus)

	w.status.WorkHeight = block.Height
	w.status.WorkHash = block.Hash()
	if w.status.WorkHeight >= w.status.BestHeight {
		w.status.BestHeight = w.status.WorkHeight
		w.status.BestHash = w.status.WorkHash
	}
	return w.commitWalletInfo(storeBatch)
}

// DetachBlock detach a block and rollback state
func (w *Wallet) DetachBlock(block *types.Block) error {
	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	storeBatch := w.DB.NewBatch()
	w.reverseAccountUTXOs(storeBatch, block, txStatus)
	w.deleteTransactions(storeBatch, w.status.BestHeight)

	w.status.BestHeight = block.Height - 1
	w.status.BestHash = block.PreviousBlockHash

	if w.status.WorkHeight > w.status.BestHeight {
		w.status.WorkHeight = w.status.BestHeight
		w.status.WorkHash = w.status.BestHash
	}

	return w.commitWalletInfo(storeBatch)
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) walletUpdater() {
	for {
		getRescanNotification(w)
		checkRescanStatus(w)
		for !w.chain.InMainChain(w.status.BestHash) {
			block, err := w.chain.GetBlockByHash(&w.status.BestHash)
			if err != nil {
				log.WithField("err", err).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.DetachBlock(block); err != nil {
				log.WithField("err", err).Error("walletUpdater detachBlock")
				return
			}
		}

		block, _ := w.chain.GetBlockByHeight(w.status.WorkHeight + 1)
		if block == nil {
			<-w.chain.BlockWaiter(w.status.WorkHeight + 1)
			continue
		}

		if err := w.AttachBlock(block); err != nil {
			log.WithField("err", err).Error("walletUpdater stop")
			return
		}
	}
}

func getRescanNotification(w *Wallet) {
	select {
	case <-w.rescanProgress:
		w.status.WorkHeight = 0
		block, _ := w.chain.GetBlockByHeight(w.status.WorkHeight)
		w.status.WorkHash = block.Hash()
	default:
		return
	}
}

// ExportAccountPrivKey exports the account private key as a WIF for encoding as a string
// in the Wallet Import Formt.
func (w *Wallet) ExportAccountPrivKey(xpub chainkd.XPub, auth string) (*string, error) {
	xprv, err := w.Hsm.LoadChainKDKey(xpub, auth)
	if err != nil {
		return nil, err
	}
	var hashed [32]byte
	sha3pool.Sum256(hashed[:], xprv[:])

	tmp := append(xprv[:], hashed[:4]...)
	res := base58.Encode(tmp)
	return &res, nil
}

// ImportAccountPrivKey imports the account key in the Wallet Import Formt.
func (w *Wallet) ImportAccountPrivKey(xprv chainkd.XPrv, keyAlias, auth string, index uint64, accountAlias string) (*pseudohsm.XPub, error) {
	if w.Hsm.HasAlias(keyAlias) {
		return nil, pseudohsm.ErrDuplicateKeyAlias
	}
	if w.Hsm.HasKey(xprv) {
		return nil, pseudohsm.ErrDuplicateKey
	}

	if acc, _ := w.AccountMgr.FindByAlias(nil, accountAlias); acc != nil {
		return nil, account.ErrDuplicateAlias
	}

	xpub, _, err := w.Hsm.ImportXPrvKey(auth, keyAlias, xprv)
	if err != nil {
		return nil, err
	}

	newAccount, err := w.AccountMgr.Create(nil, []chainkd.XPub{xpub.XPub}, SINGLE, accountAlias, nil)
	if err != nil {
		return nil, err
	}
	if err := w.recoveryAccountWalletDB(newAccount, xpub, index, keyAlias); err != nil {
		return nil, err
	}
	return xpub, nil
}

// ImportAccountXpubKey imports the account key in the Wallet Import Formt.
func (w *Wallet) ImportAccountXpubKey(xpubIndex int, xpub pseudohsm.XPub, cpIndex uint64) error {
	accountAlias := fmt.Sprintf("recovery_%d", xpubIndex)

	if acc, _ := w.AccountMgr.FindByAlias(nil, accountAlias); acc != nil {
		return account.ErrDuplicateAlias
	}

	newAccount, err := w.AccountMgr.Create(nil, []chainkd.XPub{xpub.XPub}, SINGLE, accountAlias, nil)
	if err != nil {
		return err
	}

	return w.recoveryAccountWalletDB(newAccount, &xpub, cpIndex, xpub.Alias)
}

func (w *Wallet) recoveryAccountWalletDB(account *account.Account, XPub *pseudohsm.XPub, index uint64, keyAlias string) error {
	if err := w.createProgram(account, XPub, index); err != nil {
		return err
	}
	w.ImportPrivKey = true
	tmp := KeyInfo{
		Alias:    keyAlias,
		XPub:     XPub.XPub,
		Complete: false,
	}
	w.keysInfo = append(w.keysInfo, tmp)
	w.commitkeysInfo()
	w.rescanBlocks()

	return nil
}

func (w *Wallet) createProgram(account *account.Account, XPub *pseudohsm.XPub, index uint64) error {
	for i := uint64(0); i < index; i++ {
		if _, err := w.AccountMgr.CreateAddress(nil, account.ID); err != nil {
			return err
		}
	}
	return nil
}

func (w *Wallet) rescanBlocks() {
	select {
	case w.rescanProgress <- struct{}{}:
	default:
		return
	}
}

//GetRescanStatus return key import rescan status
func (w *Wallet) GetRescanStatus() ([]KeyInfo, error) {
	keysInfo := make([]KeyInfo, len(w.keysInfo))

	if rawKeyInfo := w.DB.Get(privKeyKey); rawKeyInfo != nil {
		if err := json.Unmarshal(rawKeyInfo, &keysInfo); err != nil {
			return nil, err
		}
	}

	var status StatusInfo
	if rawWallet := w.DB.Get(walletKey); rawWallet != nil {
		if err := json.Unmarshal(rawWallet, &status); err != nil {
			return nil, err
		}
	}

	for i := range keysInfo {
		if keysInfo[i].Complete == true || status.BestHeight == 0 {
			keysInfo[i].Percent = 100
			continue
		}

		keysInfo[i].Percent = uint8(status.WorkHeight * 100 / status.BestHeight)
		if keysInfo[i].Percent == 100 {
			keysInfo[i].Complete = true
		}
	}
	return keysInfo, nil
}

//checkRescanStatus mark private key import process `Complete` if rescan finished
func checkRescanStatus(w *Wallet) {
	if !w.ImportPrivKey || w.status.WorkHeight < w.status.BestHeight {
		return
	}

	w.ImportPrivKey = false
	for _, keyInfo := range w.keysInfo {
		keyInfo.Complete = true
	}
	w.commitkeysInfo()
}
