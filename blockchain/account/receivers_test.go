package account

import (
	"context"
	"testing"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/testutil"
)

func TestCreateReceiver(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateReceiver(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateReceiver(ctx, account.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestCreateAddressReceiver(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateAddressReceiver(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateAddressReceiver(ctx, account.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
