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
	"github.com/bytom/p2p"
	log "github.com/sirupsen/logrus"
	"time"
	"sync/atomic"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing
)

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (self *SyncManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	self.fetcher.Start()
	defer self.fetcher.Stop()
	//defer self.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()

	for {
		select {
		case <-*self.newPeerCh:
			log.Info("New peer connected.")
			// Make sure we have peers to select from, then sync
			if self.sw.Peers().Size() < minDesiredPeerCount {
				break
			}
			go self.synchronise(self.blockKeeper.BestPeer())

		case <-forceSync.C:
			// Force a sync even if not enough peers are present
			go self.synchronise(self.blockKeeper.BestPeer())

		case <-self.quitSync:
			return
		}
	}
}

// synchronise tries to sync up our local block chain with a remote peer.
func (self *SyncManager) synchronise(peer *p2p.Peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}

	if self.blockKeeper.peers[peer.Key].height > self.chain.Height() {
		//self.blockKeeper.peerUpdateCh <- struct{}{}
		// Make sure only one goroutine is ever allowed past this point at once
		if !atomic.CompareAndSwapInt32(&self.synchronising, 0, 1) {
			log.Info("Synchronising ...")
			return
		}
		defer atomic.StoreInt32(&self.synchronising, 0)

		self.blockKeeper.BlockRequestWorker(peer, self.blockKeeper.peers[peer.Key].height)
	}
}
