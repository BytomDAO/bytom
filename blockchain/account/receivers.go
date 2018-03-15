package account

import (
	"context"
	"time"

	"github.com/bytom/blockchain/txbuilder"
)

const defaultReceiverExpiry = 30 * 24 * time.Hour // 30 days

// CreateAccountReceiver creates a new address receiver for an account
func (m *Manager) CreateAccountReceiver(ctx context.Context, accountInfo string) (*txbuilder.Receiver, error) {
	accountID := accountInfo
	if s, err := m.FindByAlias(ctx, accountInfo); err == nil {
		accountID = s.ID
	}

	program, err := m.CreateAddress(ctx, accountID, false)
	if err != nil {
		return nil, err
	}

	return &txbuilder.Receiver{
		ControlProgram: program.ControlProgram,
		Address:        program.Address,
	}, nil
}
