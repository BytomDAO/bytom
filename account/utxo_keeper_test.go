package account

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/testutil"
)

func TestAddUnconfirmedUtxo(t *testing.T) {
	cases := []struct {
		before   utxoKeeper
		after    utxoKeeper
		addUtxos []*UTXO
	}{
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			addUtxos: []*UTXO{},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			addUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			addUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
					bc.NewHash([32]byte{0x02}): &UTXO{OutputID: bc.NewHash([32]byte{0x02})},
					bc.NewHash([32]byte{0x03}): &UTXO{OutputID: bc.NewHash([32]byte{0x03})},
				},
			},
			addUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x02})},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03})},
			},
		},
	}

	for i, c := range cases {
		c.before.AddUnconfirmedUtxo(c.addUtxos)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestCancel(t *testing.T) {
	cases := []struct {
		before    utxoKeeper
		after     utxoKeeper
		cancelRid uint64
	}{
		{
			before: utxoKeeper{
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			cancelRid: 0,
		},
		{
			before: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			after: utxoKeeper{
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			cancelRid: 1,
		},
		{
			before: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			after: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			cancelRid: 2,
		},
		{
			before: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
					bc.NewHash([32]byte{0x02}): 3,
					bc.NewHash([32]byte{0x03}): 3,
					bc.NewHash([32]byte{0x04}): 3,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
					3: &reservation{
						id: 3,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x02})},
							&UTXO{OutputID: bc.NewHash([32]byte{0x03})},
							&UTXO{OutputID: bc.NewHash([32]byte{0x04})},
						},
						change: 9528,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			after: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			cancelRid: 3,
		},
	}

	for i, c := range cases {
		c.before.cancel(c.cancelRid)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestRemoveUnconfirmedUtxo(t *testing.T) {
	cases := []struct {
		before      utxoKeeper
		after       utxoKeeper
		removeUtxos []*bc.Hash
	}{
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			removeUtxos: []*bc.Hash{},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.Hash{V0: 1}: &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			removeUtxos: []*bc.Hash{
				&bc.Hash{V0: 1},
			},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.Hash{V0: 1}: &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
					bc.Hash{V0: 2}: &UTXO{OutputID: bc.NewHash([32]byte{0x02})},
					bc.Hash{V0: 3}: &UTXO{OutputID: bc.NewHash([32]byte{0x03})},
					bc.Hash{V0: 4}: &UTXO{OutputID: bc.NewHash([32]byte{0x04})},
					bc.Hash{V0: 5}: &UTXO{OutputID: bc.NewHash([32]byte{0x05})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.Hash{V0: 2}: &UTXO{OutputID: bc.NewHash([32]byte{0x02})},
					bc.Hash{V0: 4}: &UTXO{OutputID: bc.NewHash([32]byte{0x04})},
				},
			},
			removeUtxos: []*bc.Hash{
				&bc.Hash{V0: 1},
				&bc.Hash{V0: 3},
				&bc.Hash{V0: 5},
			},
		},
	}

	for i, c := range cases {
		c.before.RemoveUnconfirmedUtxo(c.removeUtxos)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestReserve(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		before        utxoKeeper
		after         utxoKeeper
		err           error
		reserveAmount uint64
		exp           time.Time
	}{
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				reserved:      map[bc.Hash]uint64{},
				reservations:  map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				reserved:      map[bc.Hash]uint64{},
				reservations:  map[uint64]*reservation{},
			},
			reserveAmount: 1,
			err:           ErrInsufficient,
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			reserveAmount: 4,
			err:           ErrInsufficient,
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			reserveAmount: 3,
			err:           ErrImmature,
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			reserveAmount: 3,
			err:           ErrReserved,
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x01}),
								AccountID: "testAccount",
								Amount:    3,
							},
						},
						change: 1,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveAmount: 2,
			err:           nil,
			exp:           time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				nextIndex:     1,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
					bc.NewHash([32]byte{0x02}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "testAccount",
						Amount:    5,
					},
					bc.NewHash([32]byte{0x03}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x03}),
						AccountID: "testAccount",
						Amount:    7,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
					bc.NewHash([32]byte{0x02}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "testAccount",
						Amount:    5,
					},
					bc.NewHash([32]byte{0x03}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x03}),
						AccountID: "testAccount",
						Amount:    7,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
					bc.NewHash([32]byte{0x02}): 2,
					bc.NewHash([32]byte{0x03}): 2,
				},
				reservations: map[uint64]*reservation{
					2: &reservation{
						id: 2,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x03}),
								AccountID: "testAccount",
								Amount:    7,
							},
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x02}),
								AccountID: "testAccount",
								Amount:    5,
							},
						},
						change: 4,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveAmount: 8,
			err:           nil,
			exp:           time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	for i, c := range cases {
		if _, err := c.before.Reserve("testAccount", &bc.AssetID{}, c.reserveAmount, true, c.exp); err != c.err {
			t.Errorf("case %d: got error %v want error %v", i, err, c.err)
		}
		checkUtxoKeeperEqual(t, i, &c.before, &c.after)
	}
}

func TestReserveParticular(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		before      utxoKeeper
		after       utxoKeeper
		err         error
		reserveHash bc.Hash
		exp         time.Time
	}{
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         ErrReserved,
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         ErrImmature,
		},
		{
			before: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x01}),
								AccountID: "testAccount",
								Amount:    3,
							},
						},
						change: 0,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         nil,
			exp:         time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	for i, c := range cases {
		if _, err := c.before.ReserveParticular(c.reserveHash, true, c.exp); err != c.err {
			t.Errorf("case %d: got error %v want error %v", i, err, c.err)
		}
		checkUtxoKeeperEqual(t, i, &c.before, &c.after)
	}
}

func TestExpireReservation(t *testing.T) {
	before := &utxoKeeper{
		reservations: map[uint64]*reservation{
			1: &reservation{expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC)},
			2: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			3: &reservation{expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC)},
			4: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			5: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
		},
	}
	after := &utxoKeeper{
		reservations: map[uint64]*reservation{
			2: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			4: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			5: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
		},
	}
	before.expireReservation(time.Date(2017, 8, 10, 0, 0, 0, 0, time.UTC))
	checkUtxoKeeperEqual(t, 0, before, after)
}

func TestFindUtxos(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		uk             utxoKeeper
		dbUtxos        []*UTXO
		useUnconfirmed bool
		wantUtxos      []*UTXO
		immatureAmount uint64
	}{
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos:        []*UTXO{},
			useUnconfirmed: true,
			wantUtxos:      []*UTXO{},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					AssetID:   bc.AssetID{V0: 6},
					Amount:    3,
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:    bc.NewHash([32]byte{0x02}),
					AccountID:   "testAccount",
					Amount:      3,
					ValidHeight: 9528,
				},
			},
			useUnconfirmed: false,
			wantUtxos:      []*UTXO{},
			immatureAmount: 3,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x11}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			useUnconfirmed: true,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    1,
					},
					bc.NewHash([32]byte{0x02}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "notMe",
						Amount:    2,
					},
				},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x03}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x04}),
					AccountID: "notMe",
					Amount:    4,
				},
			},
			useUnconfirmed: true,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x03}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    1,
				},
			},
			immatureAmount: 0,
		},
	}

	for i, c := range cases {
		for _, u := range c.dbUtxos {
			data, err := json.Marshal(u)
			if err != nil {
				t.Error(err)
			}
			testDB.Set(StandardUTXOKey(u.OutputID), data)
		}

		gotUtxos, immatureAmount := c.uk.findUtxos("testAccount", &bc.AssetID{}, c.useUnconfirmed)
		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
		if immatureAmount != c.immatureAmount {
			t.Errorf("case %d: got %v want %v", i, immatureAmount, c.immatureAmount)
		}

		for _, u := range c.dbUtxos {
			testDB.Delete(StandardUTXOKey(u.OutputID))
		}
	}
}

func TestFindUtxo(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		uk             utxoKeeper
		dbUtxos        map[string]*UTXO
		outHash        bc.Hash
		useUnconfirmed bool
		wantUtxo       *UTXO
		err            error
	}{
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos:  map[string]*UTXO{},
			outHash:  bc.NewHash([32]byte{0x01}),
			wantUtxo: nil,
			err:      ErrMatchUTXO,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			dbUtxos:        map[string]*UTXO{},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       nil,
			useUnconfirmed: false,
			err:            ErrMatchUTXO,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			dbUtxos:        map[string]*UTXO{},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: true,
			err:            nil,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: map[string]*UTXO{
				string(StandardUTXOKey(bc.NewHash([32]byte{0x01}))): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: false,
			err:            nil,
		},
		{
			uk: utxoKeeper{
				db:            testDB,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: map[string]*UTXO{
				string(ContractUTXOKey(bc.NewHash([32]byte{0x01}))): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: false,
			err:            nil,
		},
	}

	for i, c := range cases {
		for k, u := range c.dbUtxos {
			data, err := json.Marshal(u)
			if err != nil {
				t.Error(err)
			}
			testDB.Set([]byte(k), data)
		}

		gotUtxo, err := c.uk.findUtxo(c.outHash, c.useUnconfirmed)
		if !testutil.DeepEqual(gotUtxo, c.wantUtxo) {
			t.Errorf("case %d: got %v want %v", i, gotUtxo, c.wantUtxo)
		}
		if err != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		for _, u := range c.dbUtxos {
			testDB.Delete(StandardUTXOKey(u.OutputID))
		}
	}
}

func TestOptUTXOs(t *testing.T) {
	cases := []struct {
		uk             utxoKeeper
		input          []*UTXO
		inputAmount    uint64
		wantUtxos      []*UTXO
		optAmount      uint64
		reservedAmount uint64
	}{
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
			},
			input:          []*UTXO{},
			inputAmount:    13,
			wantUtxos:      []*UTXO{},
			optAmount:      0,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
			},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			inputAmount:    13,
			wantUtxos:      []*UTXO{},
			optAmount:      0,
			reservedAmount: 1,
		},
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
			},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
			},
			inputAmount: 13,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
			},
			optAmount:      8,
			reservedAmount: 1,
		},
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
					bc.NewHash([32]byte{0x02}): 2,
					bc.NewHash([32]byte{0x03}): 3,
				},
			},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
			},
			inputAmount:    1,
			wantUtxos:      []*UTXO{},
			optAmount:      0,
			reservedAmount: 9,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
			},
			inputAmount: 1,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      1,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
			},
			inputAmount: 5,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
			},
			optAmount:      5,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 6},
			},
			inputAmount: 6,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 6},
			},
			optAmount:      6,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 6},
			},
			inputAmount: 5,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      5,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x07}), Amount: 7},
				&UTXO{OutputID: bc.NewHash([32]byte{0x11}), Amount: 11},
				&UTXO{OutputID: bc.NewHash([32]byte{0x13}), Amount: 13},
				&UTXO{OutputID: bc.NewHash([32]byte{0x23}), Amount: 23},
				&UTXO{OutputID: bc.NewHash([32]byte{0x31}), Amount: 31},
			},
			inputAmount: 13,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x07}), Amount: 7},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
			},
			optAmount:      15,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			inputAmount: 1,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      1,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 4},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 6},
				&UTXO{OutputID: bc.NewHash([32]byte{0x07}), Amount: 7},
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 8},
				&UTXO{OutputID: bc.NewHash([32]byte{0x09}), Amount: 9},
				&UTXO{OutputID: bc.NewHash([32]byte{0x10}), Amount: 10},
				&UTXO{OutputID: bc.NewHash([32]byte{0x11}), Amount: 11},
				&UTXO{OutputID: bc.NewHash([32]byte{0x12}), Amount: 12},
			},
			inputAmount: 15,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 4},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      15,
			reservedAmount: 0,
		},
	}

	for i, c := range cases {
		got, optAmount, reservedAmount := c.uk.optUTXOs(c.input, c.inputAmount)
		if !testutil.DeepEqual(got, c.wantUtxos) {
			t.Errorf("case %d: utxos got %v want %v", i, got, c.wantUtxos)
		}
		if optAmount != c.optAmount {
			t.Errorf("case %d: utxos got %v want %v", i, optAmount, c.optAmount)
		}
		if reservedAmount != c.reservedAmount {
			t.Errorf("case %d: reservedAmount got %v want %v", i, reservedAmount, c.reservedAmount)
		}
	}
}

func checkUtxoKeeperEqual(t *testing.T, i int, a, b *utxoKeeper) {
	if !testutil.DeepEqual(a.unconfirmed, b.unconfirmed) {
		t.Errorf("case %d: unconfirmed got %v want %v", i, a.unconfirmed, b.unconfirmed)
	}
	if !testutil.DeepEqual(a.reserved, b.reserved) {
		t.Errorf("case %d: reserved got %v want %v", i, a.reserved, b.reserved)
	}
	if !testutil.DeepEqual(a.reservations, b.reservations) {
		t.Errorf("case %d: reservations got %v want %v", i, a.reservations, b.reservations)
	}
}
