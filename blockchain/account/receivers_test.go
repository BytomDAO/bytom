package account

import (
	"context"
	"testing"

	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/testutil"
)

func TestCreateAddressReceiver(t *testing.T) {
	m := mockAccountManager(t)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "test-alias", nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateAccountReceiver(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateAccountReceiver(ctx, account.Alias)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
