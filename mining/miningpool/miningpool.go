package miningpool

import (
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/event"
	"github.com/bytom/mining"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

const (
	maxSubmitChSize = 50
)

type submitBlockMsg struct {
	blockHeader *types.BlockHeader
	reply       chan error
}

// MiningPool is the support struct for p2p mine pool
type MiningPool struct {
	mutex            sync.RWMutex
	blockHeader      *types.BlockHeader
	submitCh         chan *submitBlockMsg
	commitMap        map[bc.Hash]([]*types.Tx)
	recommitInterval time.Duration

	chain           *protocol.Chain
	accountManager  *account.Manager
	txPool          *protocol.TxPool
	eventDispatcher *event.Dispatcher
}

// NewMiningPool will create a new MiningPool
func NewMiningPool(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool, dispatcher *event.Dispatcher, recommitInterval uint64) *MiningPool {
	m := &MiningPool{
		submitCh:         make(chan *submitBlockMsg, maxSubmitChSize),
		commitMap:        make(map[bc.Hash]([]*types.Tx)),
		recommitInterval: time.Duration(recommitInterval) * time.Second,
		chain:            c,
		accountManager:   accountManager,
		txPool:           txPool,
		eventDispatcher:  dispatcher,
	}
	m.generateBlock(true)
	go m.blockUpdater()
	return m
}

// blockUpdater is the goroutine for keep update mining block
func (m *MiningPool) blockUpdater() {
	recommitTicker := time.NewTicker(m.recommitInterval)
	for {
		select {
		case <-recommitTicker.C:
			m.generateBlock(false)

		case <-m.chain.BlockWaiter(m.chain.BestBlockHeight() + 1):
			m.generateBlock(true)

		case submitMsg := <-m.submitCh:
			err := m.submitWork(submitMsg.blockHeader)
			if err == nil {
				m.generateBlock(true)
			}
			submitMsg.reply <- err
		}
	}
}

// generateBlock generates a block template to mine
func (m *MiningPool) generateBlock(isNextHeight bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if isNextHeight {
		// make a new commitMap, so that the expired map will be deleted(garbage-collected)
		m.commitMap = make(map[bc.Hash]([]*types.Tx))
	}

	block, err := mining.NewBlockTemplate(m.chain, m.txPool, m.accountManager)
	if err != nil {
		log.Errorf("miningpool: failed on create NewBlockTemplate: %v", err)
		return
	}

	// The previous memory will be reclaimed by gc
	m.blockHeader = &block.BlockHeader
	m.commitMap[block.TransactionsMerkleRoot] = block.Transactions
}

// GetWork will return a block header for p2p mining
func (m *MiningPool) GetWork() (*types.BlockHeader, error) {
	if m.blockHeader != nil {
		m.mutex.RLock()
		defer m.mutex.RUnlock()

		m.blockHeader.Timestamp = uint64(time.Now().Unix())
		return m.blockHeader, nil
	}
	return nil, errors.New("no block is ready for mining")
}

// SubmitWork will try to submit the result to the blockchain
func (m *MiningPool) SubmitWork(bh *types.BlockHeader) error {
	reply := make(chan error, 1)
	m.submitCh <- &submitBlockMsg{blockHeader: bh, reply: reply}
	err := <-reply
	if err != nil {
		log.WithFields(log.Fields{"err": err, "height": bh.Height}).Warning("submitWork failed")
	}
	return err
}

func (m *MiningPool) submitWork(bh *types.BlockHeader) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.blockHeader == nil || bh.PreviousBlockHash != m.blockHeader.PreviousBlockHash {
		return errors.New("pending mining block has been changed")
	}

	txs, ok := m.commitMap[bh.TransactionsMerkleRoot]
	if !ok {
		return errors.New("TransactionsMerkleRoot not found in history")
	}

	block := &types.Block{*bh, txs}
	isOrphan, err := m.chain.ProcessBlock(block)
	if err != nil {
		return err
	}
	if isOrphan {
		return errors.New("submit result is orphan")
	}

	return m.eventDispatcher.Post(event.NewMinedBlockEvent{Block: block})
}
