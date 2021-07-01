package state

import (
	"encoding/hex"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/math/checked"
	"github.com/bytom/bytom/protocol/bc/types"
)

//  validatorRewardPerBlock the number of rewards each block validator can get
func validatorRewardPerBlock(checkpoint *Checkpoint) uint64 {
	if pledgeRate := checkpoint.pledgeRate(); pledgeRate <= consensus.RewardThreshold {
		return uint64((pledgeRate + consensus.RewardThreshold) * float64(consensus.BlockReward))
	}

	return consensus.BlockReward
}

// federationBlockReward the number of rewards each block federation can get
func federationBlockReward(checkpoint *Checkpoint) (uint64, error) {
	validatorReward := validatorRewardPerBlock(checkpoint)
	return consensus.BlockReward - validatorReward, nil
}

// pledgeRate validator's pledge rate
func (c *Checkpoint) pledgeRate() float64 {
	var totalVotes uint64
	for _, vote := range c.Votes {
		totalVotes += vote
	}

	totalSupply := c.Height*consensus.BlockReward + consensus.InitBTMSupply
	return float64(totalVotes) / float64(totalSupply)
}

// ApplyValidatorReward calculate the coinbase reward for validator
func (c *Checkpoint) ApplyValidatorReward(block *types.Block) error {
	var (
		controlProgram []byte
		feeAmount      uint64
		ok             bool
	)
	if len(block.Transactions) > 0 && len(block.Transactions[0].Outputs) > 0 {
		controlProgram = block.Transactions[0].Outputs[0].ControlProgram
	} else {
		return errors.New("not found coinbase receiver")
	}

	for _, tx := range block.Transactions {
		txFee, err := calculateTxFee(tx)
		if err != nil {
			return errors.Wrap(checked.ErrOverflow, "calculate transaction fee")
		}

		feeAmount, ok = checked.AddUint64(feeAmount, txFee)
		if !ok {
			return errors.Wrap(checked.ErrOverflow, "calculate validator reward")
		}
	}

	if c.Parent == nil {
		return errors.New("the checkpoint parent is nil")
	}

	validatorReward := validatorRewardPerBlock(c.Parent)
	validatorScript := hex.EncodeToString(controlProgram)
	c.Rewards[validatorScript] += feeAmount + validatorReward
	return nil
}

// ApplyFederationReward  federation gain the reward in an epoch
func (c *Checkpoint) ApplyFederationReward() error {
	if c.Parent == nil {
		return errors.New("the checkpoint parent is nil")
	}

	federationReward, err := federationBlockReward(c.Parent)
	if err != nil {
		return err
	}

	if federationReward == 0 {
		return nil
	}

	federationScript := config.CommonConfig.Federation.FederationScript
	c.Rewards[federationScript] = federationReward * consensus.ActiveNetParams.BlocksOfEpoch
	return nil
}

// calculateTxFee calculate transaction fee
func calculateTxFee(tx *types.Tx) (fee uint64, err error) {
	var ok bool
	for _, input := range tx.Inputs {
		if input.InputType() == types.CoinbaseInputType {
			return 0, nil
		}

		if input.AssetID() == *consensus.BTMAssetID {
			if fee, ok = checked.AddUint64(fee, input.Amount()); !ok {
				return 0, checked.ErrOverflow
			}
		}
	}

	for _, output := range tx.Outputs {
		if *output.AssetAmount.AssetId == *consensus.BTMAssetID {
			if fee, ok = checked.SubUint64(fee, output.AssetAmount.Amount); !ok {
				return 0, checked.ErrOverflow
			}
		}
	}

	return
}
