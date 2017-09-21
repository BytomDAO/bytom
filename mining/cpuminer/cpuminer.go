// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cpuminer

import (
	"fmt"
	"sync"
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/mining"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	maxNonce          = ^uint64(0) // 2^32 - 1
	defaultNumWorkers = 1
	hashUpdateSecs    = 15
)

// CPUMiner provides facilities for solving blocks (mining) using the CPU in
// a concurrency-safe manner.
type CPUMiner struct {
	sync.Mutex
	chain             *protocol.Chain
	txPool            *protocol.TxPool
	numWorkers        uint64
	started           bool
	discreteMining    bool
	wg                sync.WaitGroup
	workerWg          sync.WaitGroup
	updateNumWorkers  chan struct{}
	queryHashesPerSec chan float64
	updateHashes      chan uint64
	speedMonitorQuit  chan struct{}
	quit              chan struct{}
}

// solveBlock attempts to find some combination of a nonce, extra nonce, and
// current timestamp which makes the passed block hash to a value less than the
// target difficulty.
func (m *CPUMiner) solveBlock(block *legacy.Block, ticker *time.Ticker, quit chan struct{}) bool {
	header := &block.BlockHeader
	targetDifficulty := consensus.CompactToBig(header.Bits)

	for i := uint64(0); i <= maxNonce; i++ {
		select {
		case <-quit:
			return false

		case <-ticker.C:
			if m.chain.Height() >= header.Height {
				return false
			}
		default:
			// Non-blocking select to fall through
		}

		header.Nonce = i
		hash := header.Hash()

		// The block is solved when the new block hash is less
		// than the target difficulty.  Yay!
		//fmt.Printf("hash %v, targe %v \n ", consensus.HashToBig(&hash), targetDifficulty)
		if consensus.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
			return true
		}
	}
	return false
}

// generateBlocks is a worker that is controlled by the miningWorkerController.
// It is self contained in that it creates block templates and attempts to solve
// them while detecting when it is performing stale work and reacting
// accordingly by generating a new block template.  When a block is solved, it
// is submitted.
//
// It must be run as a goroutine.
func (m *CPUMiner) generateBlocks(quit chan struct{}) {
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

out:
	for {
		select {
		case <-quit:
			break out
		default:
		}

		//TODO: No point in searching for a solution before the chain is synced

		//TODO: get address from the wallet
		payToAddr := []byte{}

		// Create a new block template using the available transactions
		// in the memory pool as a source of transactions to potentially
		// include in the block.
		block, err := mining.NewBlockTemplate(m.chain, m.txPool, payToAddr)
		//fmt.Printf("finish to generate block template with heigh %d \n", block.BlockHeader.Height)
		if err != nil {
			fmt.Printf("Failed to create new block template: %v \n", err)
			continue
		}

		if m.solveBlock(block, ticker, quit) {
			//fmt.Printf("====================================")
			//fmt.Println(block.BlockHeader.AssetsMerkleRoot)
			snap, err := m.chain.ApplyValidBlock(block)
			if err != nil {
				fmt.Printf("Failed to apply valid block: %v \n", err)
				continue
			}
			err = m.chain.CommitAppliedBlock(nil, block, snap)
			if err != nil {
				fmt.Printf("Failed to commit block: %v \n", err)
				continue
			}
			/*fmt.Println(block)
			x, err := m.chain.GetBlock(block.BlockHeader.Height)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(x)
			fmt.Println(x == block)
			fmt.Println(block.Transactions)
			fmt.Println(x.Transactions)*/
			fmt.Printf("finish commit block heigh %d \n", block.BlockHeader.Height)
		}
	}

	m.workerWg.Done()
}

// miningWorkerController launches the worker goroutines that are used to
// generate block templates and solve them.  It also provides the ability to
// dynamically adjust the number of running worker goroutines.
//
// It must be run as a goroutine.
func (m *CPUMiner) miningWorkerController() {
	// launchWorkers groups common code to launch a specified number of
	// workers for generating blocks.
	var runningWorkers []chan struct{}
	launchWorkers := func(numWorkers uint64) {
		for i := uint64(0); i < numWorkers; i++ {
			quit := make(chan struct{})
			runningWorkers = append(runningWorkers, quit)

			m.workerWg.Add(1)
			go m.generateBlocks(quit)
		}
	}

	// Launch the current number of workers by default.
	runningWorkers = make([]chan struct{}, 0, m.numWorkers)
	launchWorkers(m.numWorkers)

out:
	for {
		select {
		// Update the number of running workers.
		case <-m.updateNumWorkers:
			// No change.
			numRunning := uint64(len(runningWorkers))
			if m.numWorkers == numRunning {
				continue
			}

			// Add new workers.
			if m.numWorkers > numRunning {
				launchWorkers(m.numWorkers - numRunning)
				continue
			}

			// Signal the most recently created goroutines to exit.
			for i := numRunning - 1; i >= m.numWorkers; i-- {
				close(runningWorkers[i])
				runningWorkers[i] = nil
				runningWorkers = runningWorkers[:i]
			}

		case <-m.quit:
			for _, quit := range runningWorkers {
				close(quit)
			}
			break out
		}
	}

	// Wait until all workers shut down to stop the speed monitor since
	// they rely on being able to send updates to it.
	m.workerWg.Wait()
	close(m.speedMonitorQuit)
	m.wg.Done()
}

// Start begins the CPU mining process as well as the speed monitor used to
// track hashing metrics.  Calling this function when the CPU miner has
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (m *CPUMiner) Start() {
	m.Lock()
	defer m.Unlock()

	// Nothing to do if the miner is already running or if running in
	// discrete mode (using GenerateNBlocks).
	if m.started || m.discreteMining {
		return
	}

	m.quit = make(chan struct{})
	m.speedMonitorQuit = make(chan struct{})
	m.wg.Add(2)
	go m.miningWorkerController()

	m.started = true
}

// Stop gracefully stops the mining process by signalling all workers, and the
// speed monitor to quit.  Calling this function when the CPU miner has not
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (m *CPUMiner) Stop() {
	m.Lock()
	defer m.Unlock()

	// Nothing to do if the miner is not currently running or if running in
	// discrete mode (using GenerateNBlocks).
	if !m.started || m.discreteMining {
		return
	}

	close(m.quit)
	m.wg.Wait()
	m.started = false
}

// IsMining returns whether or not the CPU miner has been started and is
// therefore currenting mining.
//
// This function is safe for concurrent access.
func (m *CPUMiner) IsMining() bool {
	m.Lock()
	defer m.Unlock()

	return m.started
}

// SetNumWorkers sets the number of workers to create which solve blocks.  Any
// negative values will cause a default number of workers to be used which is
// based on the number of processor cores in the system.  A value of 0 will
// cause all CPU mining to be stopped.
//
// This function is safe for concurrent access.
func (m *CPUMiner) SetNumWorkers(numWorkers int32) {
	if numWorkers == 0 {
		m.Stop()
	}

	// Don't lock until after the first check since Stop does its own
	// locking.
	m.Lock()
	defer m.Unlock()

	// Use default if provided value is negative.
	if numWorkers < 0 {
		m.numWorkers = defaultNumWorkers
	} else {
		m.numWorkers = uint64(numWorkers)
	}

	// When the miner is already running, notify the controller about the
	// the change.
	if m.started {
		m.updateNumWorkers <- struct{}{}
	}
}

// NumWorkers returns the number of workers which are running to solve blocks.
//
// This function is safe for concurrent access.
func (m *CPUMiner) NumWorkers() int32 {
	m.Lock()
	defer m.Unlock()

	return int32(m.numWorkers)
}

// New returns a new instance of a CPU miner for the provided configuration.
// Use Start to begin the mining process.  See the documentation for CPUMiner
// type for more details.
func NewCPUMiner(c *protocol.Chain, txPool *protocol.TxPool) *CPUMiner {
	return &CPUMiner{
		chain:             c,
		txPool:            txPool,
		numWorkers:        defaultNumWorkers,
		updateNumWorkers:  make(chan struct{}),
		queryHashesPerSec: make(chan float64),
		updateHashes:      make(chan uint64),
	}
}
