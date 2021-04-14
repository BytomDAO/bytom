package proposal

import (
	"sync"
	"time"

	"github.com/bytom/bytom/event"

	"github.com/bytom/bytom/errors"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/account"

	"github.com/bytom/bytom/protocol/bc"

	consensusConfig "github.com/bytom/bytom/consensus"

	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/consensus"
)

type BlockProposer struct {
	sync.Mutex
	chain           *protocol.Chain
	casper          *consensus.Casper
	accountManager  *account.Manager
	quit            chan struct{}
	started         bool
	eventDispatcher *event.Dispatcher
}

func (bp *BlockProposer) Start() {
	bp.Lock()
	defer bp.Unlock()

	if bp.started {
		return
	}
	bp.quit = make(chan struct{})
	go bp.generateBlockLoop()
	bp.started = true
}

func (bp *BlockProposer) Stop() {
	bp.Lock()
	defer bp.Unlock()

	if !bp.started {
		return
	}
	close(bp.quit)
	bp.started = false
}

func (bp *BlockProposer) generateBlockLoop() {
	ticker := time.NewTicker(time.Duration(consensusConfig.ActiveNetParams.BlockTimeInterval) * time.Millisecond)
	for {
		select {
		case <-bp.quit:
			return
		case <-ticker.C:
		}
		bp.Propose()
	}
}

func (bp *BlockProposer) Propose() error {
	_, preHash := bp.casper.BestChain()
	preHeader, _ := bp.chain.GetHeaderByHash(&preHash)

	now := uint64(time.Now().Unix())
	base := now
	if now < preHeader.Timestamp {
		base = preHeader.Timestamp
	}
	minTimeToNextBlock := consensusConfig.ActiveNetParams.BlockTimeInterval - base%consensusConfig.ActiveNetParams.BlockTimeInterval
	nextBlockTime := base + minTimeToNextBlock
	if (nextBlockTime - now) < consensusConfig.ActiveNetParams.BlockTimeInterval/10 {
		nextBlockTime += consensusConfig.ActiveNetParams.BlockTimeInterval
	}

	if !bp.inturn(preHash) {
		return errors.New("it's not your turn")
	}

	block, err := NewBlock(bp.chain, bp.casper, bp.accountManager, nextBlockTime)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on create NewBlockTemplate")
		return err
	}

	isOrphan, err := bp.chain.ProcessBlock(block)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "height": block.BlockHeader.Height, "error": err}).Error("proposer fail on ProcessBlock")
		return err
	}

	log.WithFields(log.Fields{"module": logModule, "height": block.BlockHeader.Height, "isOrphan": isOrphan, "tx": len(block.Transactions)}).Info("proposer processed block")
	if err = bp.eventDispatcher.Post(block); err != nil {
		log.WithFields(log.Fields{"module": logModule, "height": block.BlockHeader.Height, "error": err}).Error("proposer fail on post block")
	}
	return nil
}

func (bp *BlockProposer) inturn(blockHash bc.Hash) bool {

	return false
}

func (bp *BlockProposer) IsProPosing() bool {
	bp.Lock()
	defer bp.Unlock()

	return bp.started
}
