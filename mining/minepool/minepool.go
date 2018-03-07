package minepool

import (
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/mining"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

const blockUpdateMS = 1000

type MinePool struct {
	mutex sync.RWMutex
	block *legacy.Block

	chain          *protocol.Chain
	accountManager *account.Manager
	txPool         *protocol.TxPool
}

func NewMinePool(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool) *MinePool {
	m := &MinePool{
		chain:          c,
		accountManager: accountManager,
		txPool:         txPool,
	}
	go m.blockUpdater()
	return m
}

func (m *MinePool) blockUpdater() {
	ticker := time.NewTicker(time.Millisecond * blockUpdateMS)
	for _ = range ticker.C {
		m.generateBlock()
	}
}

func (m *MinePool) generateBlock() {
	if m.block != nil && m.chain.Height() < m.block.Height {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		m.block.Timestamp = uint64(time.Now().Unix())
		return
	}

	block, err := mining.NewBlockTemplate(m.chain, m.txPool, m.accountManager)
	if err != nil {
		log.Errorf("minepool: failed on create NewBlockTemplate: %v", err)
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.block = block
}

func (m *MinePool) GetWork() (*legacy.BlockHeader, error) {
	if m.block != nil {
		m.mutex.RLocker()
		defer m.mutex.RUnlock()
		return &m.block.BlockHeader, nil
	}
	return nil, errors.New("no block is ready for mining")
}

func (m *MinePool) SubmitWork(bh *legacy.BlockHeader) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.block == nil || bh.PreviousBlockHash != m.block.PreviousBlockHash {
		return false
	}

	m.block.Nonce = bh.Nonce
	m.block.Timestamp = bh.Timestamp
	_, err := m.chain.ProcessBlock(m.block)
	return err == nil
}

func (m *MinePool) CheckReward(hash *bc.Hash) (uint64, error) {
	block, err := m.chain.GetBlockByHash(hash)
	if err != nil {
		return 0, err
	}
	return block.Transactions[0].Outputs[0].Amount, nil
}
