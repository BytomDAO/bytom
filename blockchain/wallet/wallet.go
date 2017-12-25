package wallet

import (
	"encoding/json"
	"time"

	"github.com/btcsuite/btcutil/base58"
	log "github.com/sirupsen/logrus"
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
	rescanProgress chan bool
}

//NewWallet return a new wallet instance
func NewWallet(walletDB db.DB, account *account.Manager, asset *asset.Registry, chain *protocol.Chain) *Wallet {
	w := &Wallet{
		DB:         walletDB,
		AccountMgr: account,
		AssetReg:   asset,
		chain:      chain,
	}
	walletInfo, err := w.GetWalletInfo()
	if err != nil {
		log.WithField("warn", err).Warn("get wallet info")
	}
	w.status.Height = walletInfo.Height
	w.status.Hash = walletInfo.Hash
	rescanProgress := make(chan bool)
	w.rescanProgress = rescanProgress
	return w
}

//GetWalletInfo return stored wallet info and nil,if error,
//return initial wallet info and err
func (w *Wallet) GetWalletInfo() (StatusInfo, error) {
	var info StatusInfo
	var rawWallet []byte

	if rawWallet = w.DB.Get(walletKey); rawWallet == nil {
		return info, nil
	}

	if err := json.Unmarshal(rawWallet, &info); err != nil {
		return info, err
	}

	return info, nil

}

func (w *Wallet) commitWalletInfo(batch *db.Batch) error {
	var info StatusInfo

	info.Height = w.status.Height
	info.Hash = w.status.Hash

	rawWallet, err := json.Marshal(info)
	if err != nil {
		log.WithField("err", err).Error("save wallet info")
		return err
	}
	//update wallet to db
	(*batch).Set(walletKey, rawWallet)
	//commit to db
	(*batch).Write()
	return nil
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) WalletUpdate(c *protocol.Chain) {
	var err error
	var block *legacy.Block

	storeBatch := w.DB.NewBatch()

LOOP:
	getRescanNotification(w)
	for !c.InMainChain(w.status.Height, w.status.Hash) {
		if block, err = c.GetBlockByHash(&w.status.Hash); err != nil {
			log.WithField("err", err).Error("get block by hash")
			return
		}

		//Reverse this block
		reverseAccountUTXOs(&storeBatch, block, w)
		deleteTransactions(&storeBatch, w.status.Height, block, w)
		log.WithField("Height", w.status.Height).Info("start rollback this block")

		w.status.Height = block.Height - 1
		w.status.Hash = block.PreviousBlockHash

		//update wallet info and commit batch write
		if err := w.commitWalletInfo(&storeBatch); err != nil {
			return
		}
	}

	block, _ = c.GetBlockByHeight(w.status.Height + 1)
	//if we already handled the tail of the chain, we wait
	if block == nil {
		<-c.BlockWaiter(w.status.Height + 1)
		if block, err = c.GetBlockByHeight(w.status.Height + 1); err != nil {
			log.WithField("err", err).Error("wallet get block by height")
			return
		}
	}

	//if false, means that rollback operation is necessary,then goto LOOP
	if block.PreviousBlockHash == w.status.Hash {
		//next loop will save
		w.status.Height = block.Height
		w.status.Hash = block.Hash()
		indexTransactions(&storeBatch, block, w)
		buildAccountUTXOs(&storeBatch, block, w)

		//update wallet info and commit batch write
		if err := w.commitWalletInfo(&storeBatch); err != nil {
			return
		}
	}

	//goto next loop
	goto LOOP
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
	var xpubs []chainkd.XPub
	xpubs = append(xpubs, xpub.XPub)
	account, err := w.AccountMgr.Create(nil, xpubs, SINGLE, alias, nil, "")
	if err != nil {
		return nil, err
	}
	if err := recoveryAccountWalletDB(w, account, &w.DB, xpub, index); err != nil {
		return nil, err
	}
	return xpub, nil
}

func recoveryAccountWalletDB(w *Wallet, account *account.Account, DB *db.DB, XPub *pseudohsm.XPub, index uint64) error {
	if err := createProgram(w, account, DB, XPub, index); err != nil {
		return err
	}
	rescanBlocks(w)
	return nil
}

func createProgram(w *Wallet, account *account.Account, DB *db.DB, XPub *pseudohsm.XPub, index uint64) error {
	for i := uint64(0); i < index; i++ {
		_, err := w.AccountMgr.CreateControlProgram(nil, account.ID, true, time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func rescanBlocks(w *Wallet) {
	w.rescanProgress <- true
}
