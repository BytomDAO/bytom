package contract

import (
	"sync"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/asset"
	dbm "github.com/bytom/bytom/database/leveldb"
	"github.com/bytom/bytom/protocol"
	"github.com/google/uuid"
)

const (
	//SINGLE single sign
	SINGLE    = 1
	logModule = "contractview"
)

//ContractView is related to storing account unspent outputs
type ContractView struct {
	ID          uuid.UUID
	DB          dbm.DB
	rw          sync.RWMutex
	AccountMgr  *account.Manager
	AssetReg    *asset.Registry
	ContractReg *Registry
	chain       *protocol.Chain
}

//NewContratView return a new contractView instance
//TODO:set uuid && map utxo
func NewContratView(db dbm.DB, account *account.Manager, asset *asset.Registry, contract *Registry, chain *protocol.Chain) (*ContractView, error) {
	c := &ContractView{
		DB:          db,
		AccountMgr:  account,
		AssetReg:    asset,
		ContractReg: contract,
		chain:       chain,
	}

	go c.syncKeeper()
	return c, nil
}

func (c *ContractView) syncKeeper() {}

func (c *ContractView) syncUTXO() {}

func (c *ContractView) syncTransaction() {}
