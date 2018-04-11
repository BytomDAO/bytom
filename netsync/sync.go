// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package netsync

import (
	"math/rand"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/common"
	"github.com/bytom/protocol/bc/types"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing

	// This is the target size for the packs of transactions sent by txsyncLoop.
	// A pack can get larger than this if a single transactions exceeds this size.
	txsyncPackSize = 100 * 1024
)

type txsync struct {
	p   *peer
	txs []*types.Tx
}

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (sm *SyncManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	sm.fetcher.Start()
	defer sm.fetcher.Stop()
	//defer sm.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()

	for {
		select {
		case <-sm.newPeerCh:
			log.Info("New peer connected.")
			// Make sure we have peers to select from, then sync
			if sm.sw.Peers().Size() < minDesiredPeerCount {
				break
			}
			go sm.synchronise()

		case <-forceSync.C:
			// Force a sync even if not enough peers are present
			go sm.synchronise()

		case <-sm.quitSync:
			return
		}
	}
}

// synchronise tries to sync up our local block chain with a remote peer.
func (sm *SyncManager) synchronise() {
	log.Info("bk peer num:", sm.blockKeeper.peers.Len(), " sw peer num:", sm.sw.Peers().Size(), " ", sm.sw.Peers().List())
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&sm.synchronising, 0, 1) {
		log.Info("Synchronising ...")
		return
	}
	defer atomic.StoreInt32(&sm.synchronising, 0)

	peer, bestHeight := sm.peers.BestPeer()
	// Short circuit if no peers are available
	if peer == nil {
		return
	}
	if bestHeight > sm.chain.BestBlockHeight() {
		log.Info("sync peer:", peer.Addr(), " height:", bestHeight)
		sm.blockKeeper.BlockRequestWorker(peer.Key, bestHeight)
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (sm *SyncManager) txsyncLoop() {
	var (
		pending = make(map[string]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		size := common.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += common.StorageSize(s.txs[i].SerializedSize)
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.Key)
		}
		// Send the pack in the background.
		log.Info("Sending batch of transactions. ", "count:", len(pack.txs), " bytes:", size)
		sending = true
		go func() { done <- pack.p.SendTransactions(pack.txs) }()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
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
		case s := <-sm.txSyncCh:
			pending[s.p.Key] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				log.Info("Transaction send failed", "err", err)
				delete(pending, pack.p.Key)
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-sm.quitSync:
			return
		}
	}
}
