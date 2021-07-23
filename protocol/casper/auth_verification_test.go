package casper

import (
	"testing"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/database/storage"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/testutil"
)

var (
	prvKey = testutil.MustDecodeHexString("60deaf67d0bbb4d7f2e70f91eb508d87918640f0811f3a5a5f2b20246a07184bf863e6920e6ffc7a9a4ed8e43de568067ef8277fd45d3188c528a9f148f8d2f5")
	pubKey = "a2c5cbbc128485dccf41b7baec85ec9c7bd9418bc0c5ea23f4eeb621725bf9f0f863e6920e6ffc7a9a4ed8e43de568067ef8277fd45d3188c528a9f148f8d2f5"

	checkpoints = []*state.Checkpoint{
		{
			Height: 0,
			Hash:   testutil.MustDecodeHash("a1770c17493b87c43e85c2ab811023a8566907838c2237e7d72071c5f4713c5b"),
			Status: state.Justified,
			Votes: map[string]uint64{
				"a2c5cbbc128485dccf41b7baec85ec9c7bd9418bc0c5ea23f4eeb621725bf9f0f863e6920e6ffc7a9a4ed8e43de568067ef8277fd45d3188c528a9f148f8d2f5": 1e14,
				"a7ac9f585075d9f30313e290fc7a4e35e6af741cffe020e0ffd4f2e4527f0080746bd53f43d5559da7ba5b134e3f67bd851ca66d2c6955e3493abbb32c68d6fa": 1e14,
			},
		},
		{
			Height:     100,
			ParentHash: testutil.MustDecodeHash("a1770c17493b87c43e85c2ab811023a8566907838c2237e7d72071c5f4713c5b"),
			Hash:       testutil.MustDecodeHash("104da9966d3ef81d20a67a0aa79d3979b2542adeb52666deff64f63ecbfe2535"),
			Status:     state.Unjustified,
			SupLinks: []*types.SupLink{
				{
					SourceHeight: 0,
					SourceHash:   testutil.MustDecodeHash("a1770c17493b87c43e85c2ab811023a8566907838c2237e7d72071c5f4713c5b"),
					Signatures:   [consensus.MaxNumOfValidators][]byte{[]byte{0xaa}},
				},
			},
		},
		{
			Height:     100,
			ParentHash: testutil.MustDecodeHash("a1770c17493b87c43e85c2ab811023a8566907838c2237e7d72071c5f4713c5b"),
			Hash:       testutil.MustDecodeHash("ba043724330543c9b6699b62dbdc71573e666fe2caeca181e651bc96ec474b44"),
			Status:     state.Unjustified,
		},
		{
			Height:     200,
			ParentHash: testutil.MustDecodeHash("ba043724330543c9b6699b62dbdc71573e666fe2caeca181e651bc96ec474b44"),
			Hash:       testutil.MustDecodeHash("51e2a85b8e3ec0b4aabe891fca1d55a9970cf1b7b7783975bfcc169db7cf2653"),
			Status:     state.Unjustified,
		},
	}
)

func TestRollback(t *testing.T) {
	casper := NewCasper(&mockStore2{}, event.NewDispatcher(), checkpoints)
	casper.prevCheckpointCache.Add(checkpoints[1].Hash, &checkpoints[0].Hash)
	go func() {
		rollbackMsg := <-casper.rollbackCh
		if rollbackMsg.BestHash != checkpoints[1].Hash {
			t.Fatalf("want best chain %s, got %s\n", checkpoints[1].Hash.String(), rollbackMsg.BestHash.String())
		}
		rollbackMsg.Reply <- nil
	}()

	if bestHash := casper.bestChain(); bestHash != checkpoints[3].Hash {
		t.Fatalf("want best chain %s, got %s\n", checkpoints[3].Hash.String(), bestHash.String())
	}

	xPrv := chainkd.XPrv{}
	copy(xPrv[:], prvKey)
	v := &verification{
		SourceHash:   checkpoints[0].Hash,
		TargetHash:   checkpoints[1].Hash,
		SourceHeight: checkpoints[0].Height,
		TargetHeight: checkpoints[1].Height,
		PubKey:       pubKey,
	}
	if err := v.Sign(xPrv); err != nil {
		t.Fatal(err)
	}

	if err := casper.AuthVerification(&ValidCasperSignMsg{
		SourceHash: v.SourceHash,
		TargetHash: v.TargetHash,
		PubKey:     v.PubKey,
		Signature:  v.Signature,
	}); err != nil {
		t.Fatal(err)
	}

	if bestHash := casper.bestChain(); bestHash != checkpoints[1].Hash {
		t.Fatalf("want best chain %s, got %s\n", checkpoints[1].Hash.String(), bestHash.String())
	}
}

type mockStore2 struct{}

func (s *mockStore2) GetCheckpointsByHeight(u uint64) ([]*state.Checkpoint, error) { return nil, nil }
func (s *mockStore2) SaveCheckpoints([]*state.Checkpoint) error                    { return nil }
func (s *mockStore2) CheckpointsFromNode(height uint64, hash *bc.Hash) ([]*state.Checkpoint, error) {
	return nil, nil
}
func (s *mockStore2) BlockExist(hash *bc.Hash) bool                            { return false }
func (s *mockStore2) GetBlock(*bc.Hash) (*types.Block, error)                  { return nil, nil }
func (s *mockStore2) GetStoreStatus() *state.BlockStoreState                   { return nil }
func (s *mockStore2) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error { return nil }
func (s *mockStore2) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)             { return nil, nil }
func (s *mockStore2) GetMainChainHash(uint64) (*bc.Hash, error)                { return nil, nil }
func (s *mockStore2) GetContract([32]byte) ([]byte, error)                     { return nil, nil }
func (s *mockStore2) SaveBlock(*types.Block) error                             { return nil }
func (s *mockStore2) SaveBlockHeader(*types.BlockHeader) error                 { return nil }
func (s *mockStore2) SaveChainStatus(*types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, *state.ContractViewpoint, uint64, *bc.Hash) error {
	return nil
}
func (s *mockStore2) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return &types.BlockHeader{}, nil
}
func (s *mockStore2) GetCheckpoint(hash *bc.Hash) (*state.Checkpoint, error) {
	for _, c := range checkpoints {
		if c.Hash == *hash {
			return c, nil
		}
	}
	return nil, errors.New("fail to get checkpoint")
}
