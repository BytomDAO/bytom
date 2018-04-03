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
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing
)

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
	if bestHeight > sm.chain.Height() {
		sm.blockKeeper.BlockRequestWorker(peer.Key, bestHeight)
	}
}
