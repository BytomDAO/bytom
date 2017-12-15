package account

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
)

const defaultReceiverExpiry = 30 * 24 * time.Hour // 30 days

// CreateReceiver creates a new account receiver for an account
// with the provided expiry. If a zero time is provided for the
// expiry, a default expiry of 30 days from the current time is
// used.
func (m *Manager) CreateReceiver(ctx context.Context, accountInfo string, expiresAt time.Time) ([]byte, error) {
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(defaultReceiverExpiry)
	}

	accountID := accountInfo

	if s, err := m.FindByAlias(ctx, accountInfo); err == nil {
		accountID = s.ID
	}

	cp, err := m.CreateControlProgram(ctx, accountID, false, expiresAt)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	receiver := &txbuilder.Receiver{
		ControlProgram: cp,
		ExpiresAt:      expiresAt,
	}

	rawReceiver, err := json.Marshal(receiver)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return rawReceiver, nil
}
