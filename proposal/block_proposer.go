package proposal

import (
	"encoding/hex"
	"sync"
	"time"

	"github.com/bytom/bytom/config"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"

	"github.com/bytom/bytom/event"

	"github.com/bytom/bytom/errors"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/account"

	consensusConfig "github.com/bytom/bytom/consensus"

	"github.com/bytom/bytom/protocol"
)

var (
	errNotYourTurn = errors.New("not your turn to propose")
)

type BlockProposer struct {
	sync.Mutex
	chain           *protocol.Chain
	accountManager  *account.Manager
	quit            chan struct{}
	started         bool
	eventDispatcher *event.Dispatcher
}

func NewBlockProposer(chain *protocol.Chain, accountManager *account.Manager, dispatcher *event.Dispatcher) *BlockProposer {
	return &BlockProposer{
		chain:           chain,
		accountManager:  accountManager,
		eventDispatcher: dispatcher,
	}
}

func (bp *BlockProposer) Start() {
	bp.Lock()
	defer bp.Unlock()

	if bp.started {
		return
	}
	bp.quit = make(chan struct{})
	go bp.proposeLoop()
	bp.started = true
	log.Info("block proposer started")
}

func (bp *BlockProposer) Stop() {
	bp.Lock()
	defer bp.Unlock()

	if !bp.started {
		return
	}
	close(bp.quit)
	bp.started = false
	log.Info("block proposer stopped")
}

func (bp *BlockProposer) IsProPosing() bool {
	bp.Lock()
	defer bp.Unlock()

	return bp.started
}

func (bp *BlockProposer) proposeLoop() {
	ticker := time.NewTicker(time.Duration(consensusConfig.ActiveNetParams.BlockTimeInterval) * time.Millisecond)
	for {
		select {
		case <-bp.quit:
			return
		case <-ticker.C:
		}
		bp.propose()
	}
}

func (bp *BlockProposer) propose() error {
	_, preHash := bp.chain.Casper().BestChain()
	preHeader, _ := bp.chain.GetHeaderByHash(&preHash)

	blockTime := nextBlockTime(preHeader.Timestamp)
	xpubStr := getXpubStr()
	proposer, err := bp.chain.GetBlocker(&preHeader.PreviousBlockHash, blockTime)
	if err != nil {
		return err
	}
	if xpubStr != proposer {
		return errNotYourTurn
	}

	block, err := NewBlockTemplate(bp.chain, bp.accountManager, blockTime)
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

func getXpubStr() string {
	privateKeyhexStr, err := config.CommonConfig.NodeKey()
	if err != nil {
		log.WithField("err", err).Panic("fail on get private key")
	}
	var xprv chainkd.XPrv
	if _, err := hex.Decode(xprv[:], []byte(privateKeyhexStr)); err != nil {
		log.WithField("err", err).Panic("fail on decode private key")
	}
	xpub := xprv.XPub()
	xpubStr := hex.EncodeToString(xpub.Bytes())

	return xpubStr
}

func nextBlockTime(preBlockTime uint64) uint64 {
	now := uint64(time.Now().Unix() / 1e6)
	base := now
	if now < preBlockTime {
		base = preBlockTime
	}
	minTimeToNextBlock := consensusConfig.ActiveNetParams.BlockTimeInterval - base%consensusConfig.ActiveNetParams.BlockTimeInterval
	blockTime := base + minTimeToNextBlock
	if (blockTime - now) < consensusConfig.ActiveNetParams.BlockTimeInterval/10 {
		blockTime += consensusConfig.ActiveNetParams.BlockTimeInterval
	}

	return blockTime
}
