package settlementvotereward

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/jinzhu/gorm"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/toolbar/apinode"
	"github.com/bytom/bytom/toolbar/common"
	"github.com/bytom/bytom/toolbar/vote_reward/config"
)

var (
	errNotFoundReward = errors.New("No reward found")
	errNotRewardTx    = errors.New("No reward transaction")
)

type voteResult struct {
	VoteAddress string
	VoteNum     uint64
}

type SettlementReward struct {
	rewardCfg   *config.RewardConfig
	node        *apinode.Node
	db          *gorm.DB
	rewards     map[string]uint64
	startHeight uint64
	endHeight   uint64
}

type memo struct {
	StartHeight uint64 `json:"start_height"`
	EndHeight   uint64 `json:"end_height"`
	NodePubkey  string `json:"node_pubkey"`
	RewardRatio uint64 `json:"reward_ratio"`
}

func NewSettlementReward(db *gorm.DB, cfg *config.Config, startHeight, endHeight uint64) *SettlementReward {
	return &SettlementReward{
		db:          db,
		rewardCfg:   cfg.RewardConf,
		node:        apinode.NewNode(cfg.NodeIP),
		rewards:     make(map[string]uint64),
		startHeight: startHeight,
		endHeight:   endHeight,
	}
}

func (s *SettlementReward) getVoteResultFromDB(height uint64) (voteResults []*voteResult, err error) {
	query := s.db.Table("utxos").Select("vote_address, sum(vote_num) as vote_num")
	query = query.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub = ?", height-consensus.ActiveNetParams.BlocksOfEpoch+1, height-consensus.ActiveNetParams.BlocksOfEpoch, s.rewardCfg.XPub)
	query = query.Group("vote_address")
	if err := query.Scan(&voteResults).Error; err != nil {
		return nil, err
	}

	return voteResults, nil
}

func (s *SettlementReward) Settlement() error {
	for height := s.startHeight + consensus.ActiveNetParams.BlocksOfEpoch; height <= s.endHeight; height += consensus.ActiveNetParams.BlocksOfEpoch {
		totalReward, err := s.getCoinbaseReward(height + 1)
		if err == errNotFoundReward {
			continue
		}

		if err != nil {
			return errors.Wrapf(err, "get total reward at height: %d", height)
		}

		voteResults, err := s.getVoteResultFromDB(height)
		if err != nil {
			return err
		}

		s.calcVoterRewards(voteResults, totalReward)
	}

	if len(s.rewards) == 0 {
		return errNotRewardTx
	}

	data, err := json.Marshal(&memo{
		StartHeight: s.startHeight,
		EndHeight:   s.endHeight,
		NodePubkey:  s.rewardCfg.XPub,
		RewardRatio: s.rewardCfg.RewardRatio,
	})
	if err != nil {
		return err
	}

	// send transactions
	return s.node.BatchSendBTM(s.rewardCfg.AccountID, s.rewardCfg.Password, s.rewards, data)
}

func (s *SettlementReward) getCoinbaseReward(height uint64) (uint64, error) {
	block, err := s.node.GetBlockByHeight(height)
	if err != nil {
		return 0, err
	}

	miningControl, err := common.GetControlProgramFromAddress(s.rewardCfg.MiningAddress)
	if err != nil {
		return 0, err
	}

	for _, output := range block.Transactions[0].Outputs {
		if output.Amount == 0 {
			continue
		}

		if bytes.Equal(miningControl, output.ControlProgram) {
			amount := big.NewInt(0).SetUint64(output.Amount)
			rewardRatio := big.NewInt(0).SetUint64(s.rewardCfg.RewardRatio)
			amount.Mul(amount, rewardRatio).Div(amount, big.NewInt(100))

			return amount.Uint64(), nil
		}
	}
	return 0, errNotFoundReward
}

func (s *SettlementReward) calcVoterRewards(voteResults []*voteResult, totalReward uint64) {
	totalVoteNum := uint64(0)
	for _, voteResult := range voteResults {
		totalVoteNum += voteResult.VoteNum
	}

	for _, voteResult := range voteResults {
		// voteNum / totalVoteNum  * totalReward
		voteNum := big.NewInt(0).SetUint64(voteResult.VoteNum)
		total := big.NewInt(0).SetUint64(totalVoteNum)
		reward := big.NewInt(0).SetUint64(totalReward)

		amount := voteNum.Mul(voteNum, reward).Div(voteNum, total).Uint64()

		if amount != 0 {
			s.rewards[voteResult.VoteAddress] += amount
		}
	}
}
