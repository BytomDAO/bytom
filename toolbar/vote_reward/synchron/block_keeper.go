package synchron

import (
	"encoding/hex"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/toolbar/apinode"
	"github.com/bytom/bytom/toolbar/common"
	"github.com/bytom/bytom/toolbar/vote_reward/config"
	"github.com/bytom/bytom/toolbar/vote_reward/database/orm"
)

var ErrInconsistentDB = errors.New("inconsistent db status")

type ChainKeeper struct {
	db           *gorm.DB
	node         *apinode.Node
	targetHeight uint64
}

func NewChainKeeper(db *gorm.DB, cfg *config.Config, targetHeight uint64) (*ChainKeeper, error) {
	keeper := &ChainKeeper{
		db:           db,
		node:         apinode.NewNode(cfg.NodeIP),
		targetHeight: targetHeight,
	}

	chainStatus := &orm.ChainStatus{}
	if err := db.First(chainStatus).Error; err == nil {
		return keeper, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, "fail on get chainStatus")
	}

	if err := keeper.initBlockState(); err != nil {
		return nil, errors.Wrap(err, "fail on init chainStatus")
	}
	return keeper, nil
}

func (c *ChainKeeper) SyncBlock() error {
	for {
		chainStatus := &orm.ChainStatus{}
		if err := c.db.First(chainStatus).Error; err != nil {
			return errors.Wrap(err, "fail on syncBlock query chainStatus")
		}

		if chainStatus.BlockHeight >= c.targetHeight {
			break
		}

		dbTX := c.db.Begin()
		if err := c.syncChainStatus(dbTX, chainStatus); err != nil {
			dbTX.Rollback()
			return err
		}

		if err := dbTX.Commit().Error; err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainKeeper) syncChainStatus(db *gorm.DB, chainStatus *orm.ChainStatus) error {
	nextBlock, err := c.node.GetBlockByHeight(chainStatus.BlockHeight + 1)
	if err != nil {
		return err
	}

	// Normal case, the previous hash of next block equals to the hash of current block,
	// just sync to database directly.
	if nextBlock.PreviousBlockHash.String() == chainStatus.BlockHash {
		return c.AttachBlock(db, chainStatus, nextBlock)
	}

	log.WithField("block height", chainStatus.BlockHeight).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
	currentBlock, err := c.node.GetBlockByHash(chainStatus.BlockHash)
	if err != nil {
		return err
	}

	preBlock, err := c.node.GetBlockByHash(currentBlock.PreviousBlockHash.String())
	if err != nil {
		return err
	}

	return c.DetachBlock(db, chainStatus, preBlock)
}

func (c *ChainKeeper) AttachBlock(db *gorm.DB, chainStatus *orm.ChainStatus, block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if input.TypedInput.InputType() != types.VetoInputType {
				continue
			}

			outputID, err := input.SpentOutputID()
			if err != nil {
				return err
			}

			result := db.Model(&orm.Utxo{}).Where(&orm.Utxo{OutputID: outputID.String()}).Update("veto_height", block.Height)
			if err := result.Error; err != nil {
				return err
			} else if result.RowsAffected != 1 {
				return ErrInconsistentDB
			}
		}

		for i, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteOutput)
			if !ok {
				continue
			}

			utxo := &orm.Utxo{
				Xpub:        hex.EncodeToString(voteOutput.Vote),
				VoteAddress: common.GetAddressFromControlProgram(output.ControlProgram),
				VoteHeight:  block.Height,
				VoteNum:     output.Amount,
				OutputID:    tx.OutputID(i).String(),
			}

			if err := db.Save(utxo).Error; err != nil {
				return err
			}
		}
	}

	return c.updateChainStatus(db, chainStatus, block)
}

func (c *ChainKeeper) DetachBlock(db *gorm.DB, chainStatus *orm.ChainStatus, block *types.Block) error {
	if err := db.Where(&orm.Utxo{VoteHeight: chainStatus.BlockHeight}).Delete(&orm.Utxo{}).Error; err != nil {
		return err
	}

	if err := db.Model(&orm.Utxo{}).Where(&orm.Utxo{VetoHeight: chainStatus.BlockHeight}).Update("veto_height", 0).Error; err != nil {
		return err
	}

	return c.updateChainStatus(db, chainStatus, block)
}

func (c *ChainKeeper) initBlockState() error {
	block, err := c.node.GetBlockByHeight(0)
	if err != nil {
		return errors.Wrap(err, "fail on get genenis block")
	}

	blockHash := block.Hash()
	chainStatus := &orm.ChainStatus{
		BlockHeight: block.Height,
		BlockHash:   blockHash.String(),
	}
	return c.db.Save(chainStatus).Error
}

func (c *ChainKeeper) updateChainStatus(db *gorm.DB, chainStatus *orm.ChainStatus, block *types.Block) error {
	blockHash := block.Hash()
	result := db.Model(&orm.ChainStatus{}).Where(chainStatus).Updates(&orm.ChainStatus{
		BlockHeight: block.Height,
		BlockHash:   blockHash.String(),
	})
	if err := result.Error; err != nil {
		return err
	} else if result.RowsAffected != 1 {
		return ErrInconsistentDB
	}
	return nil
}
