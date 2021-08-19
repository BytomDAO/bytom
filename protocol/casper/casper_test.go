package casper

import (
	"testing"

	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/testutil"
)

func TestBestChain(t *testing.T) {
	cases := []struct {
		desc         string
		tree         *treeNode
		wantBestHash bc.Hash
	}{
		{
			desc: "only root node",
			tree: &treeNode{
				Checkpoint: &state.Checkpoint{
					Height: 0,
					Hash:   testutil.MustDecodeHash("f5d687e6a5b60fb533c4296e55260803c54e70ad4898682541da15e894971769"),
					Status: state.Justified,
				},
			},
			wantBestHash: testutil.MustDecodeHash("f5d687e6a5b60fb533c4296e55260803c54e70ad4898682541da15e894971769"),
		},
		{
			desc: "best chain is not the longest chain",
			tree: &treeNode{
				Checkpoint: &state.Checkpoint{
					Height: 0,
					Hash:   testutil.MustDecodeHash("f5d687e6a5b60fb533c4296e55260803c54e70ad4898682541da15e894971769"),
					Status: state.Finalized,
				},
				children: []*treeNode{
					{
						Checkpoint: &state.Checkpoint{
							Height: 5,
							Hash:   testutil.MustDecodeHash("750408823bea9666f263870daded75d8f3d878606ec103bc9401b73714d77729"),
							Status: state.Justified,
						},
					},
					{
						Checkpoint: &state.Checkpoint{
							Height: 5,
							Hash:   testutil.MustDecodeHash("049f012c12b94d34b13163eddbd31866f97b38c5b742e4c170a44d80ff503166"),
							Status: state.Unjustified,
						},
						children: []*treeNode{
							{
								Checkpoint: &state.Checkpoint{
									Height: 8,
									Hash:   testutil.MustDecodeHash("8315f5337978076b3097230570484e36586ee27564ebcbf74c2093cd763e32e7"),
									Status: state.Growing,
								},
							},
						},
					},
				},
			},
			wantBestHash: testutil.MustDecodeHash("750408823bea9666f263870daded75d8f3d878606ec103bc9401b73714d77729"),
		},
		{
			desc: "two distinct chain has same justified height, the longest chain is the best chain",
			tree: &treeNode{
				Checkpoint: &state.Checkpoint{
					Height: 0,
					Hash:   testutil.MustDecodeHash("f5d687e6a5b60fb533c4296e55260803c54e70ad4898682541da15e894971769"),
					Status: state.Justified,
				},
				children: []*treeNode{
					{
						Checkpoint: &state.Checkpoint{
							Height: 5,
							Hash:   testutil.MustDecodeHash("750408823bea9666f263870daded75d8f3d878606ec103bc9401b73714d77729"),
							Status: state.Unjustified,
						},
						children: []*treeNode{
							{
								Checkpoint: &state.Checkpoint{
									Height: 7,
									Hash:   testutil.MustDecodeHash("0bf26d17ff2a578c1a733a1e969184d695e8f3ac6834150acc5c1e9edeb84de9"),
									Status: state.Growing,
								},
							},
						},
					},
					{
						Checkpoint: &state.Checkpoint{
							Height: 5,
							Hash:   testutil.MustDecodeHash("049f012c12b94d34b13163eddbd31866f97b38c5b742e4c170a44d80ff503166"),
							Status: state.Unjustified,
						},
						children: []*treeNode{
							{
								Checkpoint: &state.Checkpoint{
									Height: 8,
									Hash:   testutil.MustDecodeHash("8315f5337978076b3097230570484e36586ee27564ebcbf74c2093cd763e32e7"),
									Status: state.Growing,
								},
							},
						},
					},
				},
			},
			wantBestHash: testutil.MustDecodeHash("8315f5337978076b3097230570484e36586ee27564ebcbf74c2093cd763e32e7"),
		},
	}

	for i, c := range cases {
		if bestNode, _ := c.tree.bestNode(0); bestNode.Hash != c.wantBestHash {
			t.Errorf("case #%d(%s) want best hash:%s, got best hash:%s\n", i, c.desc, c.wantBestHash.String(), bestNode.Hash.String())
		}
	}
}
