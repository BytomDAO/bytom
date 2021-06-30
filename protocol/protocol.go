package protocol

import (
	"encoding/hex"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

const (
	maxProcessBlockChSize  = 1024
	maxProcessRollbackSize = 1024
)

// Chain provides functions for working with the Bytom block chain.
type Chain struct {
	orphanManage      *OrphanManage
	txPool            *TxPool
	store             Store
	casper            *Casper
	processBlockCh    chan *processBlockMsg
	processRollbackCh chan *rollbackMsg
	eventDispatcher   *event.Dispatcher

	cond            sync.Cond
	bestBlockHeader *types.BlockHeader // the last block on current main chain
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(store Store, txPool *TxPool, eventDispatcher *event.Dispatcher) (*Chain, error) {
	return NewChainWithOrphanManage(store, txPool, NewOrphanManage(), eventDispatcher)
}

func NewChainWithOrphanManage(store Store, txPool *TxPool, manage *OrphanManage, eventDispatcher *event.Dispatcher) (*Chain, error) {
	c := &Chain{
		orphanManage:      manage,
		eventDispatcher:   eventDispatcher,
		txPool:            txPool,
		store:             store,
		processRollbackCh: make(chan *rollbackMsg, maxProcessRollbackSize),
		processBlockCh:    make(chan *processBlockMsg, maxProcessBlockChSize),
	}
	c.cond.L = new(sync.Mutex)

	storeStatus := store.GetStoreStatus()
	if storeStatus == nil {
		if err := c.initChainStatus(); err != nil {
			return nil, err
		}
		storeStatus = store.GetStoreStatus()
	}

	var err error
	c.bestBlockHeader, err = c.store.GetBlockHeader(storeStatus.Hash)
	if err != nil {
		return nil, err
	}

	casper, err := newCasper(store, storeStatus, c.processRollbackCh)
	if err != nil {
		return nil, err
	}

	c.casper = casper
	go c.blockProcessor()
	return c, nil
}

func (c *Chain) initChainStatus() error {
	genesisBlock := config.GenesisBlock()
	if err := c.store.SaveBlock(genesisBlock); err != nil {
		return err
	}

	checkpoint := &state.Checkpoint{
		Height:    0,
		Hash:      genesisBlock.Hash(),
		Timestamp: genesisBlock.Timestamp,
		Status:    state.Justified,
	}

	if err := c.store.SaveCheckpoints([]*state.Checkpoint{checkpoint}); err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	bcBlock := types.MapBlock(genesisBlock)
	if err := utxoView.ApplyBlock(bcBlock); err != nil {
		return err
	}

	contractView := state.NewContractViewpoint()
	genesisBlockHeader := &genesisBlock.BlockHeader
	return c.store.SaveChainStatus(genesisBlockHeader, []*types.BlockHeader{genesisBlockHeader}, utxoView, contractView, 0, &checkpoint.Hash)
}

func newCasper(store Store, storeStatus *BlockStoreState, rollbackCh chan *rollbackMsg) (*Casper, error) {
	checkpoints, err := store.CheckpointsFromNode(storeStatus.FinalizedHeight, storeStatus.FinalizedHash)
	if err != nil {
		return nil, err
	}

	return NewCasper(store, checkpoints, rollbackCh), nil
}

// LastJustifiedHeader return the last justified block header of the block chain
func (c *Chain) LastJustifiedHeader() (*types.BlockHeader, error) {
	_, hash := c.casper.LastJustified()
	return c.store.GetBlockHeader(&hash)
}

// LastFinalizedHeader return the last finalized block header of the block chain
func (c *Chain) LastFinalizedHeader() (*types.BlockHeader, error) {
	_, hash := c.casper.LastFinalized()
	return c.store.GetBlockHeader(&hash)
}

// ProcessBlockVerification process block verification
func (c *Chain) ProcessBlockVerification(v *Verification) error {
	if err := c.casper.AuthVerification(v); err != nil {
		return err
	}

	pubKey, _ := hex.DecodeString(v.PubKey)
	return c.eventDispatcher.Post(event.BlockVerificationEvent{
		SourceHeight: v.SourceHeight,
		SourceHash:   v.SourceHash,
		TargetHeight: v.TargetHeight,
		TargetHash:   v.TargetHash,
		PubKey:       pubKey,
		Signature:    v.Signature,
	})
}

// BestBlockHeight returns the current height of the blockchain.
func (c *Chain) BestBlockHeight() uint64 {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestBlockHeader.Height
}

// BestBlockHash return the hash of the chain tail block
func (c *Chain) BestBlockHash() *bc.Hash {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	bestHash := c.bestBlockHeader.Hash()
	return &bestHash
}

// AllValidators return all validators has vote num
func (c *Chain) AllValidators(blockHash *bc.Hash) ([]*state.Validator, error) {
	 parentCheckpoint, err := c.casper.parentCheckpoint(blockHash)
	 if err != nil {
	 	return nil, err
	 }

	 return parentCheckpoint.AllValidators(), nil
}

// GetValidator return validator by specified blockHash and timestamp
func (c *Chain) GetValidator(prevHash *bc.Hash, timeStamp uint64) (*state.Validator, error) {
	parentCheckpoint, err := c.casper.parentCheckpointByPrevHash(prevHash)
	if err != nil {
		return nil, err
	}

	validators := parentCheckpoint.EffectiveValidators()
	startTimestamp := parentCheckpoint.Timestamp + consensus.ActiveNetParams.BlockTimeInterval
	order := getValidatorOrder(startTimestamp, timeStamp, uint64(len(validators)))
	for _, validator := range validators {
		if validator.Order == int(order) {
			return validator, nil
		}
	}
	return nil, errors.New("get blocker failure")
}

func getValidatorOrder(startTimestamp, blockTimestamp, numOfValidators uint64) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := numOfValidators * consensus.ActiveNetParams.BlockTimeInterval
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (blockTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// Order of blocker
	return (blockTimestamp - lastRoundStartTime) / consensus.ActiveNetParams.BlockTimeInterval
}

// BestBlockHeader returns the chain tail block
func (c *Chain) BestBlockHeader() *types.BlockHeader {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestBlockHeader
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(hash bc.Hash) bool {
	blockHeader, err := c.store.GetBlockHeader(&hash)
	if err != nil {
		return false
	}

	blockHash, err := c.store.GetMainChainHash(blockHeader.Height)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "height": blockHeader.Height}).Debug("not contain block hash in main chain for specified height")
		return false
	}
	return *blockHash == hash
}

func (c *Chain) SignBlockHeader(blockHeader *types.BlockHeader) {
	xprv := config.CommonConfig.PrivateKey()
	signature := xprv.Sign(blockHeader.Hash().Bytes())
	blockHeader.Set(signature)
}

// This function must be called with mu lock in above level
func (c *Chain) setState(blockHeader *types.BlockHeader, mainBlockHeaders []*types.BlockHeader, view *state.UtxoViewpoint, contractView *state.ContractViewpoint) error {
	finalizedHeight, finalizedHash := c.casper.LastFinalized()
	if err := c.store.SaveChainStatus(blockHeader, mainBlockHeaders, view, contractView, finalizedHeight, &finalizedHash); err != nil {
		return err
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	c.bestBlockHeader = blockHeader

	hash := c.bestBlockHeader.Hash()
	log.WithFields(log.Fields{"module": logModule, "height": c.bestBlockHeader.Height, "hash": hash.String()}).Debug("chain best status has been update")
	c.cond.Broadcast()
	return nil
}

// BlockWaiter returns a channel that waits for the block at the given height.
func (c *Chain) BlockWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.cond.L.Lock()
		defer c.cond.L.Unlock()
		for c.bestBlockHeader.Height < height {
			c.cond.Wait()
		}
		ch <- struct{}{}
	}()

	return ch
}

// GetTxPool return chain txpool.
func (c *Chain) GetTxPool() *TxPool {
	return c.txPool
}

// PrevCheckpointByPrevHash get previous checkpoint by previous block hash
func (c *Chain) PrevCheckpointByPrevHash(preBlockHash *bc.Hash) (*state.Checkpoint, error) {
	return c.casper.parentCheckpointByPrevHash(preBlockHash)
}
