package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
	"github.com/bytom/protocol/validation"
)

var (
	// ErrBadBlock is returned when a block is invalid.
	ErrBadBlock = errors.New("invalid block")

	// ErrBadStateRoot is returned when the computed assets merkle root
	// disagrees with the one declared in a block header.
	ErrBadStateRoot = errors.New("invalid state merkle root")
)

// BlockExist check is a block in chain or orphan
func (c *Chain) BlockExist(hash *bc.Hash) bool {
	return c.orphanManage.BlockExist(hash) || c.store.BlockExist(hash)
}

// GetBlockByHash return a block by given hash
func (c *Chain) GetBlockByHash(hash *bc.Hash) (*legacy.Block, error) {
	return c.store.GetBlock(hash)
}

// GetBlockByHeight return a block by given height
func (c *Chain) GetBlockByHeight(height uint64) (*legacy.Block, error) {
	c.state.cond.L.Lock()
	hash, ok := c.state.mainChain[height]
	c.state.cond.L.Unlock()
	if !ok {
		return nil, errors.New("can't find block in given hight")
	}
	return c.GetBlockByHash(hash)
}

// ConnectBlock append block to end of chain
func (c *Chain) ConnectBlock(block *legacy.Block) error {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.connectBlock(block)
}

func (c *Chain) connectBlock(block *legacy.Block) (err error) {
	bcBlock := legacy.MapBlock(block)
	utxoView := state.NewUtxoViewpoint()
	bcBlock.TransactionStatus, err = c.store.GetTransactionStatus(&bcBlock.ID)
	if err != nil {
		return err
	}

	if err := c.store.GetTransactionsUtxo(utxoView, bcBlock.Transactions); err != nil {
		return err
	}
	if err := utxoView.ApplyBlock(bcBlock, bcBlock.TransactionStatus); err != nil {
		return err
	}

	blockHash := block.Hash()
	if err := c.setState(block, utxoView, map[uint64]*bc.Hash{block.Height: &blockHash}); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) getReorganizeBlocks(block *legacy.Block) ([]*legacy.Block, []*legacy.Block) {
	attachBlocks := []*legacy.Block{}
	detachBlocks := []*legacy.Block{}
	ancestor := block

	for !c.inMainchain(ancestor) {
		attachBlocks = append([]*legacy.Block{ancestor}, attachBlocks...)
		ancestor, _ = c.GetBlockByHash(&ancestor.PreviousBlockHash)
	}

	for d := c.state.block; d.Hash() != ancestor.Hash(); d, _ = c.GetBlockByHash(&d.PreviousBlockHash) {
		detachBlocks = append(detachBlocks, d)
	}

	return attachBlocks, detachBlocks
}

func (c *Chain) reorganizeChain(block *legacy.Block) error {
	attachBlocks, detachBlocks := c.getReorganizeBlocks(block)
	utxoView := state.NewUtxoViewpoint()
	chainChanges := map[uint64]*bc.Hash{}

	for _, d := range detachBlocks {
		detachBlock := legacy.MapBlock(d)
		if err := c.store.GetTransactionsUtxo(utxoView, detachBlock.Transactions); err != nil {
			return err
		}
		txStatus, err := c.GetTransactionStatus(&detachBlock.ID)
		if err != nil {
			return err
		}
		if err := utxoView.DetachBlock(detachBlock, txStatus); err != nil {
			return err
		}
	}

	for _, a := range attachBlocks {
		attachBlock := legacy.MapBlock(a)
		if err := c.store.GetTransactionsUtxo(utxoView, attachBlock.Transactions); err != nil {
			return err
		}
		txStatus, err := c.GetTransactionStatus(&attachBlock.ID)
		if err != nil {
			return err
		}

		if err := utxoView.ApplyBlock(attachBlock, txStatus); err != nil {
			return err
		}
		chainChanges[a.Height] = &attachBlock.ID
	}

	return c.setState(block, utxoView, chainChanges)
}

// SaveBlock will validate and save block into storage
func (c *Chain) SaveBlock(block *legacy.Block) error {
	preBlock, _ := c.GetBlockByHash(&block.PreviousBlockHash)
	blockEnts := legacy.MapBlock(block)
	prevEnts := legacy.MapBlock(preBlock)

	seed, err := c.GetSeed(block.Height, &block.PreviousBlockHash)
	if err != nil {
		return err
	}

	if err := validation.ValidateBlock(blockEnts, prevEnts, seed); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	if err := c.store.SaveBlock(block, blockEnts.TransactionStatus, seed); err != nil {
		return err
	}

	blockHash := block.Hash()
	log.WithFields(log.Fields{"height": block.Height, "hash": blockHash.String()}).Info("Block saved on disk")
	return nil
}

func (c *Chain) findBestChainTail(block *legacy.Block) (bestBlock *legacy.Block) {
	bestBlock = block
	blockHash := block.Hash()
	preorphans, ok := c.orphanManage.preOrphans[blockHash]
	if !ok {
		return
	}

	for _, preorphan := range preorphans {
		orphanBlock, ok := c.orphanManage.Get(preorphan)
		if !ok {
			continue
		}

		if err := c.SaveBlock(orphanBlock); err != nil {
			log.WithFields(log.Fields{
				"height": block.Height,
				"hash":   blockHash.String(),
			}).Errorf("findBestChainTail fail on save block %v", err)
			continue
		}

		if subResult := c.findBestChainTail(orphanBlock); subResult.Height > bestBlock.Height {
			bestBlock = subResult
		}
	}

	c.orphanManage.Delete(&blockHash)
	return
}

// ProcessBlock is the entry for handle block insert
func (c *Chain) ProcessBlock(block *legacy.Block) (bool, error) {
	blockHash := block.Hash()
	if c.BlockExist(&blockHash) {
		log.WithField("hash", blockHash.String()).Info("Skip process due to block already been handled")
		return false, nil
	}
	if !c.store.BlockExist(&block.PreviousBlockHash) {
		c.orphanManage.Add(block)
		return true, nil
	}
	if err := c.SaveBlock(block); err != nil {
		return false, err
	}

	bestBlock := c.findBestChainTail(block)
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	if c.state.block.Hash() == bestBlock.PreviousBlockHash {
		return false, c.connectBlock(bestBlock)
	}

	if bestBlock.Height > c.state.block.Height && bestBlock.Bits >= c.state.block.Bits {
		return false, c.reorganizeChain(bestBlock)
	}

	return false, nil
}
