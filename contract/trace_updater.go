package contract

import (
	"github.com/bytom/bytom/protocol"
	log "github.com/sirupsen/logrus"
)

var logModule = "tracer"

type TraceUpdater struct {
	*TraceService
	chain *protocol.Chain
}

func NewTraceUpdater(traceService *TraceService, chain *protocol.Chain) *TraceUpdater {
	return &TraceUpdater{
		TraceService: traceService,
		chain:        chain,
	}
}

func (t *TraceUpdater) Sync() {
	for {
		for !t.chain.InMainChain(t.BestHash()) {
			block, err := t.chain.GetBlockByHash(&t.bestHash)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := t.DetachBlock(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater detachBlock stop")
				return
			}
		}

		block, _ := t.chain.GetBlockByHeight(t.BestHeight() + 1)
		if block == nil {
			t.walletBlockWaiter()
			continue
		}

		if err := t.ApplyBlock(block); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater AttachBlock stop")
			return
		}
	}
}

func (t *TraceUpdater) walletBlockWaiter() {
	<-t.chain.BlockWaiter(t.bestHeight + 1)
}
