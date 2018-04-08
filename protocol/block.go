package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
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
	return c.orphanManage.BlockExist(hash) || c.index.BlockExist(hash)
}

// GetBlockByHash return a block by given hash
func (c *Chain) GetBlockByHash(hash *bc.Hash) (*types.Block, error) {
	return c.store.GetBlock(hash)
}

// GetBlockByHeight return a block by given height
func (c *Chain) GetBlockByHeight(height uint64) (*types.Block, error) {
	node := c.index.NodeByHeight(height)
	if node == nil {
		return nil, errors.New("can't find block in given hight")
	}
	return c.store.GetBlock(&node.Hash)
}

// ConnectBlock append block to end of chain
func (c *Chain) ConnectBlock(block *types.Block) error {
	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()
	return c.connectBlock(block)
}

func (c *Chain) connectBlock(block *types.Block) (err error) {
	bcBlock := types.MapBlock(block)
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

	if err := c.setState(block, utxoView); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) getReorganizeBlocks(block *types.Block) ([]*types.Block, []*types.Block) {
	attachBlocks := []*types.Block{}
	detachBlocks := []*types.Block{}
	ancestor := block

	for !c.index.InMainchain(block.Hash()) {
		attachBlocks = append([]*types.Block{ancestor}, attachBlocks...)
		ancestor, _ = c.GetBlockByHash(&ancestor.PreviousBlockHash)
	}

	for d, _ := c.store.GetBlock(c.state.hash); d.Hash() != ancestor.Hash(); d, _ = c.store.GetBlock(&d.PreviousBlockHash) {
		detachBlocks = append(detachBlocks, d)
	}

	return attachBlocks, detachBlocks
}

func (c *Chain) reorganizeChain(block *types.Block) error {
	attachBlocks, detachBlocks := c.getReorganizeBlocks(block)
	utxoView := state.NewUtxoViewpoint()

	for _, d := range detachBlocks {
		detachBlock := types.MapBlock(d)
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
		log.WithFields(log.Fields{"height": detachBlock.Height, "hash": detachBlock.ID.String()}).Debug("Detach from mainchain")
	}

	for _, a := range attachBlocks {
		attachBlock := types.MapBlock(a)
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
		log.WithFields(log.Fields{"height": attachBlock.Height, "hash": attachBlock.ID.String()}).Debug("Attach from mainchain")
	}

	return c.setState(block, utxoView)
}

// SaveBlock will validate and save block into storage
func (c *Chain) SaveBlock(block *types.Block) error {
	blockEnts := types.MapBlock(block)
	if err := c.validateBlock(blockEnts); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	if err := c.store.SaveBlock(block, blockEnts.TransactionStatus); err != nil {
		return err
	}
	log.WithFields(log.Fields{"height": block.Height, "hash": blockEnts.ID.String()}).Info("Block saved on disk")

	c.orphanManage.Delete(&blockEnts.ID)
	parent := c.index.GetNode(&block.PreviousBlockHash)
	node, err := NewBlockNode(&block.BlockHeader, parent)
	if err != nil {
		return err
	}

	c.index.AddNode(node)
	return nil
}

func (c *Chain) findBestChainTail(block *types.Block) (bestBlock *types.Block) {
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

	return
}

type processBlockResponse struct {
	isOrphan bool
	err      error
}

type processBlockMsg struct {
	block *types.Block
	reply chan processBlockResponse
}

func (c *Chain) ProcessBlock(block *types.Block) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	c.processBlockCh <- &processBlockMsg{block: block, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

func (c *Chain) blockProcesser() {
	for msg := range c.processBlockCh {
		isOrphan, err := c.processBlock(msg.block)
		msg.reply <- processBlockResponse{isOrphan: isOrphan, err: err}
	}
}

// ProcessBlock is the entry for handle block insert
func (c *Chain) processBlock(block *types.Block) (bool, error) {
	blockHash := block.Hash()
	if c.BlockExist(&blockHash) {
		log.WithField("hash", blockHash.String()).Debug("Skip process due to block already been handled")
		return c.orphanManage.BlockExist(&blockHash), nil
	}
	if !c.store.BlockExist(&block.PreviousBlockHash) {
		log.WithField("hash", blockHash.String()).Debug("Add block to orphan manage")
		c.orphanManage.Add(block)
		return true, nil
	}
	if err := c.SaveBlock(block); err != nil {
		return false, err
	}

	bestBlock := c.findBestChainTail(block)
	bestMainChain := c.index.BestNode()
	bestBlockHash := bestBlock.Hash()
	bestNode := c.index.GetNode(&bestBlockHash)

	if bestNode.parent == bestMainChain {
		log.WithField("hash", blockHash.String()).Debug("Start to append block to the tail of mainchain")
		return false, c.connectBlock(bestBlock)
	}

	if bestNode.height > bestMainChain.height && bestNode.workSum.Cmp(bestMainChain.workSum) >= 0 {
		log.WithField("hash", blockHash.String()).Debug("Start to reorganize mainchain")
		return false, c.reorganizeChain(bestBlock)
	}

	return false, nil
}
