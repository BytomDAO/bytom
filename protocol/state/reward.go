package state

import (
	"encoding/hex"

	"github.com/bytom/bytom/config"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/math/checked"
	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	totalSupply       = 15.66 * 1e16
	singleBlockReward = uint64(570776255) //AnnualSupply(0.3 * 1e16) / AnnualBlock(365 * 24 * 60 * 10)
	rewardThreshold   = 0.5
)

//  validatorRewardPerBlock the number of rewards each block validator can get
func validatorRewardPerBlock(checkpoint *Checkpoint) (uint64, error) {
	pledgeRate, err := pledgeRate(checkpoint)
	if err != nil {
		return 0, nil
	}

	if pledgeRate <= rewardThreshold {
		return uint64((pledgeRate + rewardThreshold) * float64(singleBlockReward)), nil
	}

	return singleBlockReward, nil
}

// federationBlockReward the number of rewards each block federation can get
func federationBlockReward(checkpoint *Checkpoint) (uint64, error) {
	validatorReward, err := validatorRewardPerBlock(checkpoint)
	if err != nil {
		return 0, err
	}

	return singleBlockReward - validatorReward, nil
}

// pledgeRate validator's pledge rate
func pledgeRate(checkpoint *Checkpoint) (float64, error) {
	var totalVotes uint64
	var ok bool
	for _, vote := range checkpoint.Votes {
		if totalVotes, ok = checked.AddUint64(totalVotes, vote); !ok {
			return 0.0, errors.Wrap(checked.ErrOverflow)
		}
	}

	for _, guaranty := range checkpoint.Guaranties {
		if totalVotes, ok = checked.AddUint64(totalVotes, guaranty); !ok {
			return 0.0, errors.Wrap(checked.ErrOverflow)
		}
	}

	if totalVotes > totalSupply {
		return 0.0, errors.New("validators total votes exceed total supply")
	}

	return float64(totalVotes) / totalSupply, nil
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

	validatorReward, err := validatorRewardPerBlock(c.Parent)
	if err != nil {
		return err
	}

	validatorScript := hex.EncodeToString(controlProgram)
	c.Rewards[validatorScript] += feeAmount + validatorReward
	return nil
}

// ApplyFederationReward  federation gain the reward in an epoch
func (c *Checkpoint) ApplyFederationReward() error {
	federationReward, err := federationBlockReward(c.Parent)
	if err != nil {
		return err
	}

	if federationReward == 0 {
		return nil
	}

	federationScript := config.CommonConfig.Federation.FederationScript
	c.Rewards[federationScript] = federationReward * BlocksOfEpoch
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
