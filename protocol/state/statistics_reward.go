package state

import (
	"encoding/hex"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/math/checked"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

type RewardAndProgram struct {
	reward         uint64
	controlProgram []byte
}

// StatisticsReward
type StatisticsReward struct {
	rewards     map[string]uint64
	BlockHash   bc.Hash
	BlockHeight uint64
}

func (sr *StatisticsReward) AttachBlock(block *types.Block) error {
	if block.PreviousBlockHash != sr.BlockHash {
		return errors.New("block previous hash is not equal to reward hash")
	}

	if block.Height%BlocksOfEpoch == 1 {
		sr.rewards = map[string]uint64{}
	}

	if err := sr.calculateReward(block, true); err != nil {
		return err
	}

	sr.BlockHash = block.Hash()
	sr.BlockHeight++
	return nil
}

func (sr *StatisticsReward) calculateReward(block *types.Block, isAdd bool) error {
	blockReward, err := calculateReward(block)
	if err != nil {
		return err
	}

	hexControlProgram := hex.EncodeToString(blockReward.controlProgram)
	if isAdd {
		sr.rewards[hexControlProgram] += blockReward.reward
	} else {
		sr.rewards[hexControlProgram] -= blockReward.reward
	}

	return nil
}

func (sr *StatisticsReward) DetachBlock(block *types.Block) error {
	if block.Hash() != sr.BlockHash {
		return errors.New("the block %s is not exist in reward")
	}

	if err := sr.calculateReward(block, false); err != nil {
		return err
	}

	sr.BlockHash = block.PreviousBlockHash
	sr.BlockHeight--
	return nil
}

func (sr *StatisticsReward) getRewards() (rewards []RewardAndProgram) {
	for hexProgram, rewardAmount := range sr.rewards {
		program, _ := hex.DecodeString(hexProgram)
		rewards = append(rewards, RewardAndProgram{
			reward:         rewardAmount,
			controlProgram: program,
		})
	}

	return
}

func calculateReward(block *types.Block) (RewardAndProgram, error) {
	var rp RewardAndProgram
	if txs := block.Transactions; len(txs) > 0 {
		if len(txs[0].Outputs) == 0 {
			return rp, errors.New("not found coinbase receiver")
		}

		rp.controlProgram = txs[0].Outputs[0].ControlProgram
	} else {
		return rp, errors.New("not found coinbase receiver")
	}

	rp.reward = consensus.BlockSubsidy(block.Height, 100)
	for _, tx := range block.Transactions {
		var fee uint64
		var err error
		if fee, err = calculateFee(tx); err != nil {
			return rp, errors.Wrap(checked.ErrOverflow, "calculate transaction fee")
		}

		rp.reward += fee
	}

	return rp, nil
}

func calculateFee(tx *types.Tx) (uint64, error) {
	var fee uint64
	var ok bool
	for _, input := range tx.Inputs {
		if input.TypedInput.InputType() == types.CoinbaseInputType {
			continue
		}
		if fee, ok = checked.AddUint64(input.Amount(), fee); !ok {
			return 0, checked.ErrOverflow
		}
	}
	for _, output := range tx.Outputs {
		if fee, ok = checked.SubUint64(fee, output.Amount); !ok {
			return 0, checked.ErrOverflow
		}
	}

	return fee, nil
}
