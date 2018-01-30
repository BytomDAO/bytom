package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/go-wire/data/base58"
	"github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

//SINGLE single sign
const SINGLE = 1

var walletKey = []byte("walletInfo")
var privKeyKey = []byte("keysInfo")

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	Height uint64
	Hash   bc.Hash
}

//KeyInfo is key import status
type KeyInfo struct {
	Alias      string       `json:"alias"`
	XPub       chainkd.XPub `json:"xpub"`
	ImportFlag bool         `json:"flag"`
	Percent    uint8        `json:"percent"`
	Complete   bool         `json:"complete"`
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB             db.DB
	status         StatusInfo
	AccountMgr     *account.Manager
	AssetReg       *asset.Registry
	chain          *protocol.Chain
	rescanProgress chan struct{}
	ImportPrivKey  bool
	keysInfo       []KeyInfo
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, account *account.Manager, asset *asset.Registry, chain *protocol.Chain) (*Wallet, error) {
	w := &Wallet{
		DB:             walletDB,
		AccountMgr:     account,
		AssetReg:       asset,
		chain:          chain,
		rescanProgress: make(chan struct{}, 1),
		ImportPrivKey:  false,
		keysInfo:       make([]KeyInfo, 0),
	}

	if err := w.loadWalletInfo(); err != nil {
		return nil, err
	}

	if err := w.loadKeysInfo(); err != nil {
		return nil, err
	}
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
	return w.attachBlock(block)
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

func (w *Wallet) attachBlock(block *legacy.Block) error {
	if block.PreviousBlockHash != w.status.Hash {
		log.Warn("wallet skip attachBlock due to status hash not equal to previous hash")
		return nil
	}

	storeBatch := w.DB.NewBatch()
	w.indexTransactions(storeBatch, block)
	w.buildAccountUTXOs(storeBatch, block)

	w.status.Height = block.Height
	w.status.Hash = block.Hash()
	return w.commitWalletInfo(storeBatch)
}

func (w *Wallet) detachBlock(block *legacy.Block) error {
	storeBatch := w.DB.NewBatch()
	w.reverseAccountUTXOs(storeBatch, block)
	w.deleteTransactions(storeBatch, w.status.Height)

	w.status.Height = block.Height - 1
	w.status.Hash = block.PreviousBlockHash
	return w.commitWalletInfo(storeBatch)
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) walletUpdater() {
	for {
		getRescanNotification(w)
		checkRescanStatus(w)
		for !w.chain.InMainChain(w.status.Height, w.status.Hash) {
			block, err := w.chain.GetBlockByHash(&w.status.Hash)
			if err != nil {
				log.WithField("err", err).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.detachBlock(block); err != nil {
				log.WithField("err", err).Error("walletUpdater detachBlock")
				return
			}
		}
		block, _ := w.chain.GetBlockByHeight(w.status.Height + 1)
		if block == nil {
			<-w.chain.BlockWaiter(w.status.Height + 1)
			continue
		}

		if err := w.attachBlock(block); err != nil {
			log.WithField("err", err).Error("walletUpdater stop")
			return
		}
	}
}

func getRescanNotification(w *Wallet) {
	select {
	case <-w.rescanProgress:
		w.status.Height = 1
		block, _ := w.chain.GetBlockByHeight(w.status.Height)
		w.status.Hash = block.Hash()
	default:
		return
	}
}

// ExportAccountPrivKey exports the account private key as a WIF for encoding as a string
// in the Wallet Import Formt.
func (w *Wallet) ExportAccountPrivKey(hsm *pseudohsm.HSM, xpub chainkd.XPub, auth string) (*string, error) {
	xprv, err := hsm.LoadChainKDKey(xpub, auth)
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
func (w *Wallet) ImportAccountPrivKey(hsm *pseudohsm.HSM, xprv chainkd.XPrv, keyAlias, auth string, index uint64, accountAlias string) (*pseudohsm.XPub, error) {
	xpub, _, err := hsm.ImportXPrvKey(auth, keyAlias, xprv)
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

func (w *Wallet) recoveryAccountWalletDB(account *account.Account, XPub *pseudohsm.XPub, index uint64, keyAlias string) error {
	if err := w.createProgram(account, XPub, index); err != nil {
		return err
	}
	w.ImportPrivKey = true
	tmp := KeyInfo{
		Alias:      keyAlias,
		XPub:       XPub.XPub,
		ImportFlag: true,
		Complete:   false,
	}
	w.keysInfo = append(w.keysInfo, tmp)
	w.commitkeysInfo()
	w.rescanBlocks()

	return nil
}

func (w *Wallet) createProgram(account *account.Account, XPub *pseudohsm.XPub, index uint64) error {
	for i := uint64(0); i < index; i++ {
		if _, err := w.AccountMgr.CreateAddress(nil, account.ID, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *Wallet) rescanBlocks() {
	w.rescanProgress <- struct{}{}
}

//GetRescanStatus return key import rescan status
func (w *Wallet) GetRescanStatus() ([]KeyInfo, error) {
	if err := w.loadKeysInfo(); err != nil {
		return nil, err
	}
	for i, v := range w.keysInfo {
		if v.ImportFlag == true {
			w.keysInfo[i].Percent = uint8(w.status.Height * 100 / w.chain.Height())
		}
	}
	return w.keysInfo, nil
}

func checkRescanStatus(w *Wallet) {
	if w.ImportPrivKey {
		if w.status.Height == w.chain.Height() {
			w.ImportPrivKey = false
			for i := range w.keysInfo {
				w.keysInfo[i].Complete = true
			}
		}

		w.commitkeysInfo()
	}
}
