package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
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
	return c.index.BlockExist(hash) || c.orphanManage.BlockExist(hash)
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

func (c *Chain) connectBlock(block *types.Block) (err error) {
	bcBlock := types.MapBlock(block)
	if bcBlock.TransactionStatus, err = c.store.GetTransactionStatus(&bcBlock.ID); err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	if err := c.store.GetTransactionsUtxo(utxoView, bcBlock.Transactions); err != nil {
		return err
	}
	if err := utxoView.ApplyBlock(bcBlock, bcBlock.TransactionStatus); err != nil {
		return err
	}

	node := c.index.GetNode(&bcBlock.ID)
	if err := c.setState(node, utxoView); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) calcReorganizeNodes(node *state.BlockNode) ([]*state.BlockNode, []*state.BlockNode) {
	var attachNodes []*state.BlockNode
	var detachNodes []*state.BlockNode

	attachIter := node
	for c.index.NodeByHeight(attachIter.Height) != attachIter {
		attachNodes = append([]*state.BlockNode{attachIter}, attachNodes...)
		attachIter = attachIter.Parent
	}

	detachIter := c.bestNode
	for detachIter != attachIter {
		detachNodes = append(detachNodes, detachIter)
		detachIter = detachIter.Parent
	}
	return attachNodes, detachNodes
}

func (c *Chain) reorganizeChain(node *state.BlockNode) error {
	attachNodes, detachNodes := c.calcReorganizeNodes(node)
	utxoView := state.NewUtxoViewpoint()

	for _, detachNode := range detachNodes {
		b, err := c.store.GetBlock(&detachNode.Hash)
		if err != nil {
			return err
		}

		detachBlock := types.MapBlock(b)
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

		log.WithFields(log.Fields{"height": node.Height, "hash": node.Hash.String()}).Debug("detach from mainchain")
	}

	for _, attachNode := range attachNodes {
		b, err := c.store.GetBlock(&attachNode.Hash)
		if err != nil {
			return err
		}

		attachBlock := types.MapBlock(b)
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

		log.WithFields(log.Fields{"height": node.Height, "hash": node.Hash.String()}).Debug("attach from mainchain")
	}

	return c.setState(node, utxoView)
}

// SaveBlock will validate and save block into storage
func (c *Chain) saveBlock(block *types.Block) error {
	bcBlock := types.MapBlock(block)
	parent := c.index.GetNode(&block.PreviousBlockHash)

	if err := validation.ValidateBlock(bcBlock, parent); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}
	if err := c.store.SaveBlock(block, bcBlock.TransactionStatus); err != nil {
		return err
	}

	c.orphanManage.Delete(&bcBlock.ID)
	node, err := state.NewBlockNode(&block.BlockHeader, parent)
	if err != nil {
		return err
	}

	c.index.AddNode(node)
	return nil
}

func (c *Chain) saveSubBlock(block *types.Block) *types.Block {
	blockHash := block.Hash()
	prevOrphans, ok := c.orphanManage.GetPrevOrphans(&blockHash)
	if !ok {
		return block
	}

	bestBlock := block
	for _, prevOrphan := range prevOrphans {
		orphanBlock, ok := c.orphanManage.Get(prevOrphan)
		if !ok {
			log.WithFields(log.Fields{"hash": prevOrphan.String()}).Warning("saveSubBlock fail to get block from orphanManage")
			continue
		}
		if err := c.saveBlock(orphanBlock); err != nil {
			log.WithFields(log.Fields{"hash": prevOrphan.String(), "height": orphanBlock.Height}).Warning("saveSubBlock fail to save block")
			continue
		}

		if subBestBlock := c.saveSubBlock(orphanBlock); subBestBlock.Height > bestBlock.Height {
			bestBlock = subBestBlock
		}
	}
	return bestBlock
}

type processBlockResponse struct {
	isOrphan bool
	err      error
}

type processBlockMsg struct {
	block *types.Block
	reply chan processBlockResponse
}

// ProcessBlock is the entry for chain update
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
		log.WithFields(log.Fields{"hash": blockHash.String(), "height": block.Height}).Info("block has been processed")
		return c.orphanManage.BlockExist(&blockHash), nil
	}

	if parent := c.index.GetNode(&block.PreviousBlockHash); parent == nil {
		c.orphanManage.Add(block)
		return true, nil
	}

	if err := c.saveBlock(block); err != nil {
		return false, err
	}

	bestBlock := c.saveSubBlock(block)
	bestBlockHash := bestBlock.Hash()
	bestNode := c.index.GetNode(&bestBlockHash)

	if bestNode.Parent == c.bestNode {
		log.Debug("append block to the end of mainchain")
		return false, c.connectBlock(bestBlock)
	}

	if bestNode.Height > c.bestNode.Height && bestNode.WorkSum.Cmp(c.bestNode.WorkSum) >= 0 {
		log.Debug("start to reorganize chain")
		return false, c.reorganizeChain(bestNode)
	}
	return false, nil
}
