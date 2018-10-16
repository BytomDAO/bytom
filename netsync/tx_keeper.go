package netsync

import (
	"math/rand"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc/types"
)

const (
	// This is the target size for the packs of transactions sent by txSyncLoop.
	// A pack can get larger than this if a single transactions exceeds this size.
	txSyncPackSize = 100 * 1024
)

type txSyncMsg struct {
	peerID string
	txs    []*types.Tx
}

func (sm *SyncManager) syncTransactions(peerID string) {
	pending := sm.txPool.GetTransactions()
	if len(pending) == 0 {
		return
	}

	txs := make([]*types.Tx, len(pending))
	for i, batch := range pending {
		txs[i] = batch.Tx
	}
	sm.txSyncCh <- &txSyncMsg{peerID, txs}
}

func (sm *SyncManager) txBroadcastLoop() {
	for {
		select {
		case newTx := <-sm.newTxCh:
			if err := sm.peers.broadcastTx(newTx); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on broadcast new tx.")
				return
			}
		case <-sm.quitSync:
			return
		}
	}
}

// txSyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (sm *SyncManager) txSyncLoop() {
	pending := make(map[string]*txSyncMsg)
	sending := false            // whether a send is active
	done := make(chan error, 1) // result of the send

	// send starts a sending a pack of transactions from the sync.
	send := func(msg *txSyncMsg) {
		peer := sm.peers.getPeer(msg.peerID)
		if peer == nil {
			delete(pending, msg.peerID)
			return
		}

		totalSize := uint64(0)
		sendTxs := []*types.Tx{}
		for i := 0; i < len(msg.txs) && totalSize < txSyncPackSize; i++ {
			sendTxs = append(sendTxs, msg.txs[i])
			totalSize += msg.txs[i].SerializedSize
		}

		if len(msg.txs) == len(sendTxs) {
			delete(pending, msg.peerID)
		} else {
			msg.txs = msg.txs[len(sendTxs):]
		}

		// Send the pack in the background.
		log.WithFields(log.Fields{
			"module": logModule,
			"count":  len(sendTxs),
			"bytes":  totalSize,
			"peer":   msg.peerID,
		}).Debug("txSyncLoop sending transactions")
		sending = true
		go func() {
			ok, err := peer.sendTransactions(sendTxs)
			if !ok {
				sm.peers.removePeer(msg.peerID)
			}
			done <- err
		}()
	}

	// pick chooses the next pending sync.
	pick := func() *txSyncMsg {
		if len(pending) == 0 {
			return nil
		}

		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case msg := <-sm.txSyncCh:
			pending[msg.peerID] = msg
			if !sending {
				send(msg)
			}

		case err := <-done:
			sending = false
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on txSyncLoop sending")
			}

			if s := pick(); s != nil {
				send(s)
			}
		}
	}
}
