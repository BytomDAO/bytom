package blockchain

import (
	"context"
	"encoding/json"

	"github.com/bytom/errors"
)

func (bcr *BlockchainReactor) GetTxFeedByAlias(ctx context.Context, filter string) ([]byte, error) {
	jf, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	value := bcr.TxFeedTracker.DB.Get(jf)
	if value == nil {
		return nil, errors.New("No transaction feed")
	}

	return value, nil
}
