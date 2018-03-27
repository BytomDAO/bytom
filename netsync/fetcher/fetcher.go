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

// Package fetcher contains the block announcement based synchronisation.
package fetcher

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	arriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested
	gatherSlack   = 100 * time.Millisecond // Interval used to collate almost-expired announces with fetches
	maxUncleDist  = 7                      // Maximum allowed backward distance from the chain head
	maxQueueDist  = 1024                   //32 // Maximum allowed distance from the chain head to queue
	hashLimit     = 256                    // Maximum number of unique blocks a peer may have announced
	blockLimit    = 64                     // Maximum number of unique blocks a peer may have delivered
)

var (
	errTerminated = errors.New("terminated")
)

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
type blockRetrievalFn func(hash *bc.Hash) (*legacy.Block, error)

// blockBroadcasterFn is a callback type for broadcasting a block to connected peers.
type blockBroadcasterFn func(block *legacy.Block)

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() uint64

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
type chainInsertFn func(block *legacy.Block) (bool, error)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// inject represents a schedules import operation.
type inject struct {
	origin string
	block  *legacy.Block
}

// Fetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
type Fetcher struct {
	// Various event channels
	inject chan *inject
	//done   chan bc.Hash
	quit   chan struct{}

	// Block cache
	queue  *prque.Prque        // Queue containing the import operations (block number sorted)
	queues map[string]int      // Per peer block counts to prevent memory exhaustion
	queued map[bc.Hash]*inject // Set of already queued blocks (to dedup imports)

	// Callbacks
	getBlock       blockRetrievalFn   // Retrieves a block from the local chain
	broadcastBlock blockBroadcasterFn // Broadcasts a block to connected peers
	chainHeight    chainHeightFn      // Retrieves the current chain's height
	insertChain    chainInsertFn      // Injects a batch of blocks into the chain
	dropPeer       peerDropFn         // Drops a peer for misbehaving
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(getBlock blockRetrievalFn /*verifyHeader headerVerifierFn,*/, broadcastBlock blockBroadcasterFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	return &Fetcher{
		inject:         make(chan *inject),
		//done:           make(chan bc.Hash),
		quit:           make(chan struct{}),
		queue:          prque.New(),
		queues:         make(map[string]int),
		queued:         make(map[bc.Hash]*inject),
		getBlock:       getBlock,
		broadcastBlock: broadcastBlock,
		chainHeight:    chainHeight,
		insertChain:    insertChain,
		dropPeer:       dropPeer,
	}
}

// Start boots up the announcement based synchroniser, accepting and processing
// hash notifications and block fetches until termination requested.
func (f *Fetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *Fetcher) Stop() {
	close(f.quit)
}

// Enqueue tries to fill gaps the the fetcher's future import queue.
func (f *Fetcher) Enqueue(peer string, block *legacy.Block) error {
	op := &inject{
		origin: peer,
		block:  block,
	}
	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Loop is the main fetcher loop, checking and processing various notification
// events.
func (f *Fetcher) loop() {
	for {
		// Import any queued blocks that could potentially fit
		height := f.chainHeight()
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)
			// If too high up the chain or phase, continue later
			number := op.block.Height
			if number > height+1 {
				f.queue.Push(op, -float32(op.block.Height))
				break
			}
			// Otherwise if fresh and still unknown, try and import
			hash := op.block.Hash()
			block, _ := f.getBlock(&hash)
			if number+maxUncleDist < height || block != nil {
				f.forgetBlock(hash)
				continue
			}
			f.insert(op.origin, op.block)
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case op := <-f.inject:
			// A direct block insertion was requested, try and fill any pending gaps
			f.enqueue(op.origin, op.block)
		}
	}
}

// enqueue schedules a new future import operation, if the block to be imported
// has not yet been seen.
func (f *Fetcher) enqueue(peer string, block *legacy.Block) {
	hash := block.Hash()

	//// Ensure the peer isn't DOSing us
	//count := f.queues[peer] + 1
	//if count > blockLimit {
	//	log.Info("Discarded propagated block, exceeded allowance", "peer", peer, "number", block.Height, "hash", hash, "limit", blockLimit)
	//	//f.forgetHash(hash)
	//	return
	//}
	// Discard any past or too distant blocks
	if dist := int64(block.Height) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
		log.Info("Discarded propagated block, too far away", "peer", peer, "number", block.Height, "hash", hash, "distance", dist)
		//f.forgetHash(hash)
		return
	}
	// Schedule the block for future importing
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin: peer,
			block:  block,
		}
		//fmt.Println("block count:", count)
		//f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(block.Height))
		log.Debug("Queued propagated block", "peer", peer, "number", block.Height, "hash", hash, "queued", f.queue.Size())
	}
}

// insert spawns a new goroutine to run a block insertion into the chain. If the
// block's number is at the same height as the current import phase, it updates
// the phase states accordingly.
func (f *Fetcher) insert(peer string, block *legacy.Block) {
	hash := block.Hash()

	// Run the import on a new thread
	log.Info("Importing propagated block", "peer", peer, "number", block.Height, "hash", hash)
	go func() {
		//defer func() { f.done <- hash }()

		// If the parent's unknown, abort insertion
		parent, err := f.getBlock(&block.PreviousBlockHash)
		if err != nil {
			log.Info("Unknown parent of propagated block", "peer", peer, "number", block.Height, "hash", hash, "parent", block.PreviousBlockHash.String())
			return
		}
		if parent == nil {
			log.Info("Unknown parent of propagated block", "peer", peer, "number", block.Height, "hash", hash, "parent", block.PreviousBlockHash.String())
			return
		}

		// Run the actual import and log any issues
		if _, err := f.insertChain(block); err != nil {
			log.Info("Propagated block import failed", "peer", peer, "number", block.Height, "hash", hash, "err", err)
			return
		}
		// If import succeeded, broadcast the block
		log.Info("success insert block from cache. height:", block.Height)
		//TODO: broadcastBlock result check
		go f.broadcastBlock(block)
	}()
}

// forgetBlock removes all traces of a queued block from the fetcher's internal
// state.
func (f *Fetcher) forgetBlock(hash bc.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}
