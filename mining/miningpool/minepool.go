package miningpool

import (
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/mining"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
)

const blockUpdateMS = 1000

// MiningPool is the support struct for p2p mine pool
type MiningPool struct {
	mutex sync.RWMutex
	block *legacy.Block

	chain          *protocol.Chain
	accountManager *account.Manager
	txPool         *protocol.TxPool
}

// NewMiningPool will create a new MiningPool
func NewMiningPool(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool) *MiningPool {
	m := &MiningPool{
		chain:          c,
		accountManager: accountManager,
		txPool:         txPool,
	}
	go m.blockUpdater()
	return m
}

// blockUpdater is the goroutine for keep update mining block
func (m *MiningPool) blockUpdater() {
	ticker := time.NewTicker(time.Millisecond * blockUpdateMS)
	for _ = range ticker.C {
		m.generateBlock()
	}
}

func (m *MiningPool) generateBlock() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.block != nil && *m.chain.BestBlockHash() == m.block.PreviousBlockHash {
		m.block.Timestamp = uint64(time.Now().Unix())
		return
	}

	block, err := mining.NewBlockTemplate(m.chain, m.txPool, m.accountManager)
	if err != nil {
		log.Errorf("miningpool: failed on create NewBlockTemplate: %v", err)
		return
	}

	m.block = block
}

// GetWork will return a block header for p2p mining
func (m *MiningPool) GetWork() (*legacy.BlockHeader, error) {
	if m.block != nil {
		m.mutex.RLock()
		defer m.mutex.RUnlock()
		bh := m.block.BlockHeader
		return &bh, nil
	}
	return nil, errors.New("no block is ready for mining")
}

// SubmitWork will try to submit the result to the blockchain
func (m *MiningPool) SubmitWork(bh *legacy.BlockHeader) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.block == nil || bh.PreviousBlockHash != m.block.PreviousBlockHash {
		return false
	}

	m.block.Nonce = bh.Nonce
	m.block.Timestamp = bh.Timestamp
	isOrphan, err := m.chain.ProcessBlock(m.block)

	if err != nil {
		log.Errorf("fail on SubmitWork on ProcessBlock %v", err)
	} else if isOrphan {
		log.Warning("SubmitWork is orphan")
	}
	return err == nil
}
