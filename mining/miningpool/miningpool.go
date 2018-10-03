package miningpool

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/mining"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	maxSubmitChSize = 50
)

type submitBlockMsg struct {
	block *types.Block
	reply chan error
}

type submitWorkMsg struct {
	blockHeader *types.BlockHeader
	reply       chan error
}

// MiningPool is the support struct for p2p mine pool
type MiningPool struct {
	mutex         sync.RWMutex
	block         *types.Block
	submitBlockCh chan *submitBlockMsg
	submitWorkCh  chan *submitWorkMsg

	chain          *protocol.Chain
	accountManager *account.Manager
	txPool         *protocol.TxPool
	newBlockCh     chan *bc.Hash
}

// NewMiningPool will create a new MiningPool
func NewMiningPool(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool, newBlockCh chan *bc.Hash) *MiningPool {
	m := &MiningPool{
		submitBlockCh:  make(chan *submitBlockMsg, maxSubmitChSize),
		submitWorkCh:   make(chan *submitWorkMsg, maxSubmitChSize),
		chain:          c,
		accountManager: accountManager,
		txPool:         txPool,
		newBlockCh:     newBlockCh,
	}
	m.generateBlock()
	go m.blockUpdater()
	return m
}

// blockUpdater is the goroutine for keep update mining block
func (m *MiningPool) blockUpdater() {
	for {
		select {
		case <-m.chain.BlockWaiter(m.chain.BestBlockHeight() + 1):
			m.generateBlock()

		case submitBlockMsg := <-m.submitBlockCh:
			err := m.submitBlock(submitBlockMsg.block)
			if err == nil {
				m.generateBlock()
			}
			submitBlockMsg.reply <- err
		}

		case submitWorkMsg := <-m.submitWorkCh:
			err := m.submitWork(submitWorkMsg.blockHeader)
			if err == nil {
				m.generateBlock()
			}
			submitWorkMsg.reply <- err
	}
}

// generateBlock generates a block template to mine
func (m *MiningPool) generateBlock() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	block, err := mining.NewBlockTemplate(m.chain, m.txPool, m.accountManager)
	if err != nil {
		log.Errorf("miningpool: failed on create NewBlockTemplate: %v", err)
		return
	}
	m.block = block
}

// GetWork will return a block header for p2p mining
func (m *MiningPool) GetWork() (*types.BlockHeader, error) {
	if m.block != nil {
		m.mutex.RLock()
		defer m.mutex.RUnlock()
		bh := m.block.BlockHeader
		return &bh, nil
	}
	return nil, errors.New("no block is ready for mining")
}

// SubmitWork will try to submit a raw block to the blockchain
func (m *MiningPool) SubmitBlock(b *types.Block) error {
	reply := make(chan error, 1)
	m.submitBlockCh <- &submitBlockMsg{block: b, reply: reply}
	err := <-reply
	if err != nil {
		log.WithFields(log.Fields{"err": err, "height": b.Height}).Warning("submitBlock failed")
	}
	return err
}

func (m *MiningPool) submitBlock(b *types.Block) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.block == nil || b.PreviousBlockHash != m.block.PreviousBlockHash {
		return errors.New("pending mining block has been changed")
	}

	m.block = b
	isOrphan, err := m.chain.ProcessBlock(m.block)
	if err != nil {
		return err
	}
	if isOrphan {
		return errors.New("submit block: submit result is orphan")
	}

	blockHash := b.BlockHeader.Hash()
	m.newBlockCh <- &blockHash
	return nil
}

// SubmitWork will try to submit a work to the blockchain
func (m *MiningPool) SubmitWork(bh *types.BlockHeader) error {
	reply := make(chan error, 1)
	m.submitWorkCh <- &submitWorkMsg{blockHeader: bh, reply: reply}
	err := <-reply
	if err != nil {
		log.WithFields(log.Fields{"err": err, "height": bh.Height}).Warning("submitWork failed")
	}
	return err
}

func (m *MiningPool) submitWork(bh *types.BlockHeader) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.block == nil || bh.PreviousBlockHash != m.block.PreviousBlockHash {
		return errors.New("pending mining block has been changed")
	}

	m.block.Nonce = bh.Nonce
	m.block.Timestamp = bh.Timestamp
	isOrphan, err := m.chain.ProcessBlock(m.block)
	if err != nil {
		return err
	}
	if isOrphan {
		return errors.New("submit work: submit result is orphan")
	}

	blockHash := bh.Hash()
	m.newBlockCh <- &blockHash
	return nil
}
