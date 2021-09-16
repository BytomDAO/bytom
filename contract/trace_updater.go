package contract

import (
	log "github.com/sirupsen/logrus"
)

var logModule = "tracer"

type TraceUpdater struct {
	*TraceService
	chain ChainService
}

func NewTraceUpdater(traceService *TraceService, chain ChainService) *TraceUpdater {
	return &TraceUpdater{
		TraceService: traceService,
		chain:        chain,
	}
}

func (t *TraceUpdater) Sync() {
	for {
		block, _ := t.chain.GetBlockByHeight(t.BestHeight() + 1)
		if block == nil {
			t.walletBlockWaiter()
			continue
		}

		if bestHash := t.BestHash(); block.PreviousBlockHash != bestHash {
			block, err := t.chain.GetBlockByHash(&bestHash)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err, "block_hash": bestHash.String()}).Errorf("trace updater get block")
				return
			}

			if err := t.DetachBlock(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("trace updater detach block")
				return
			}
		}

		if err := t.ApplyBlock(block); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("trace updater attach block")
			return
		}
	}
}

func (t *TraceUpdater) walletBlockWaiter() {
	<-t.chain.BlockWaiter(t.bestHeight + 1)
}
