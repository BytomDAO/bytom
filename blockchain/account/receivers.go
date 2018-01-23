package account

import (
	"context"
	"time"

	"encoding/hex"
	"github.com/bytom/blockchain/txbuilder"
)

const defaultReceiverExpiry = 30 * 24 * time.Hour // 30 days

// CreateAddressReceiver creates a new address receiver for an account
func (m *Manager) CreateAddressReceiver(ctx context.Context, accountInfo string) (*txbuilder.Receiver, error) {
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

// CreatePubkeyInfo creates a new public key for an account
func (m *Manager) CreatePubkeyInfo(ctx context.Context, accountInfo string) (*txbuilder.AccountPubkey, error) {
	accountID := accountInfo
	if s, err := m.FindByAlias(ctx, accountInfo); err == nil {
		accountID = s.ID
	}

	rootXPub, pubkey, path, err := m.createPubkey(ctx, accountID)
	if err != nil {
		return nil, err
	}

	var pathStr []string
	for _, p := range path {
		pathStr = append(pathStr, hex.EncodeToString(p))
	}

	return &txbuilder.AccountPubkey{
		Root:   rootXPub,
		Pubkey: hex.EncodeToString(pubkey),
		Path:   pathStr,
	}, nil
}

// CreateContractInfo creates a relationship to link contract with an account
func (m *Manager) CreateContractInfo(ctx context.Context, accountInfo string, contractProgram string) (map[string]string, error) {
	accountID := accountInfo
	if s, err := m.FindByAlias(ctx, accountInfo); err == nil {
		accountID = s.ID
	}

	contract, err := m.CreateContractHook(ctx, accountID, contractProgram)
	if err != nil {
		return nil, err
	}

	return map[string]string{"contract_program": hex.EncodeToString(contract)}, nil
}
