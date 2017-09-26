package prottest

import (
	"sync"
	"testing"

	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/state"
	"github.com/bytom/testutil"
)

var (
	mutex         sync.Mutex // protects the following
	states        = make(map[*protocol.Chain]*state.Snapshot)
	blockPubkeys  = make(map[*protocol.Chain][]ed25519.PublicKey)
	blockPrivkeys = make(map[*protocol.Chain][]ed25519.PrivateKey)
)

type Option func(testing.TB, *config)

func WithStore(store protocol.Store) Option {
	return func(_ testing.TB, conf *config) { conf.store = store }
}

func WithOutputIDs(outputIDs ...bc.Hash) Option {
	return func(_ testing.TB, conf *config) {
		for _, oid := range outputIDs {
			conf.initialState.Tree.Insert(oid.Bytes())
		}
	}
}

func WithBlockSigners(quorum, n int) Option {
	return func(tb testing.TB, conf *config) {
		conf.quorum = quorum
		for i := 0; i < n; i++ {
			pubkey, privkey, err := ed25519.GenerateKey(nil)
			if err != nil {
				testutil.FatalErr(tb, err)
			}
			conf.pubkeys = append(conf.pubkeys, pubkey)
			conf.privkeys = append(conf.privkeys, privkey)
		}
	}
}

type config struct {
	store        protocol.Store
	initialState *state.Snapshot
	pubkeys      []ed25519.PublicKey
	privkeys     []ed25519.PrivateKey
	quorum       int
}

// Initial returns the provided Chain's initial block.
func Initial(tb testing.TB, c *protocol.Chain) *legacy.Block {
	b1, err := c.GetBlock(1)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return b1
}

// BlockKeyPairs returns the configured block-signing key-pairs
// for the provided Chain.
func BlockKeyPairs(c *protocol.Chain) ([]ed25519.PublicKey, []ed25519.PrivateKey) {
	mutex.Lock()
	defer mutex.Unlock()
	return blockPubkeys[c], blockPrivkeys[c]
}
