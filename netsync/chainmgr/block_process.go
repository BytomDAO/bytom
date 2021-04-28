package chainmgr

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/netsync/peers"
	"github.com/bytom/bytom/p2p/security"
)

var errOrphanBlock = errors.New("fast sync inserting orphan block")

type blockProcessor struct {
	chain   Chain
	storage *storage
	peers   *peers.PeerSet
}

func newBlockProcessor(chain Chain, storage *storage, peers *peers.PeerSet) *blockProcessor {
	return &blockProcessor{
		chain:   chain,
		peers:   peers,
		storage: storage,
	}
}

func (bp *blockProcessor) insert(blockStorage *blockStorage) error {
	isOrphan, err := bp.chain.ProcessBlock(blockStorage.block)
	if isOrphan {
		bp.peers.ProcessIllegal(blockStorage.peerID, security.LevelMsgIllegal, errOrphanBlock.Error())
		return errOrphanBlock
	}

	if err != nil {
		bp.peers.ProcessIllegal(blockStorage.peerID, security.LevelMsgIllegal, err.Error())
	}
	return err
}

func (bp *blockProcessor) process(downloadNotifyCh chan struct{}, ProcessStop chan struct{}, syncHeight uint64, wg *sync.WaitGroup) {
	defer func() {
		close(ProcessStop)
		wg.Done()
	}()

	for {
		for {
			block, err := bp.storage.readBlock(syncHeight)
			if err != nil {
				break
			}

			if err := bp.insert(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on process block")
				return
			}

			bp.storage.deleteBlock(syncHeight)
			syncHeight++
		}

		if _, ok := <-downloadNotifyCh; !ok {
			return
		}
	}
}
