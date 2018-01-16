package wallet

import (
	"encoding/json"
	"time"

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

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	Height uint64
	Hash   bc.Hash
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	DB             db.DB
	status         StatusInfo
	AccountMgr     *account.Manager
	AssetReg       *asset.Registry
	chain          *protocol.Chain
	rescanProgress chan struct{}
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, account *account.Manager, asset *asset.Registry, chain *protocol.Chain) (*Wallet, error) {
	w := &Wallet{
		DB:             walletDB,
		AccountMgr:     account,
		AssetReg:       asset,
		chain:          chain,
		rescanProgress: make(chan struct{}, 1),
	}

	if err := w.loadWalletInfo(); err != nil {
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
	if err := w.attachBlock(block); err != nil {
		return err
	}
	return nil
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
func (w *Wallet) ImportAccountPrivKey(hsm *pseudohsm.HSM, xprv chainkd.XPrv, alias, auth string, index uint64) (*pseudohsm.XPub, error) {
	xpub, _, err := hsm.ImportXPrvKey(auth, alias, xprv)
	if err != nil {
		return nil, err
	}
	newAccount, err := w.AccountMgr.Create([]chainkd.XPub{xpub.XPub}, SINGLE, alias, nil)
	if err != nil {
		return nil, err
	}
	if err := w.recoveryAccountWalletDB(newAccount, xpub, index); err != nil {
		return nil, err
	}
	return xpub, nil
}

func (w *Wallet) recoveryAccountWalletDB(account *account.Account, XPub *pseudohsm.XPub, index uint64) error {
	if err := w.createProgram(account, XPub, index); err != nil {
		return err
	}
	w.rescanBlocks()

	return nil
}

func (w *Wallet) createProgram(account *account.Account, XPub *pseudohsm.XPub, index uint64) error {
	for i := uint64(0); i < index; i++ {
		if _, err := w.AccountMgr.CreateControlProgram(nil, account.ID, true, time.Now()); err != nil {
			return err
		}
	}
	return nil
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) rescanBlocks() {
	w.rescanProgress <- struct{}{}
}
