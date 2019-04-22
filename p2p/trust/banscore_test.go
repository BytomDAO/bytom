package trust

import (
	"math"
	"testing"
	"time"
)

func TestInt(t *testing.T) {
	var banScoreIntTests = []struct {
		bs        DynamicBanScore
		timeLapse int64
		wantValue uint32
	}{
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, timeLapse: 1, wantValue: 99},
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, timeLapse: Lifetime, wantValue: 50},
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, timeLapse: Lifetime + 1, wantValue: 50},
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, timeLapse: -1, wantValue: 50},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: 0}, timeLapse: Lifetime + 1, wantValue: 0},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: math.MaxUint32}, timeLapse: 0, wantValue: math.MaxUint32},
		{bs: DynamicBanScore{lastUnix: 0, transient: math.MaxUint32, persistent: 0}, timeLapse: Lifetime + 1, wantValue: 0},
		{bs: DynamicBanScore{lastUnix: 0, transient: math.MaxUint32, persistent: 0}, timeLapse: 60, wantValue: math.MaxUint32 / 2},
		{bs: DynamicBanScore{lastUnix: 0, transient: math.MaxUint32, persistent: math.MaxUint32}, timeLapse: 0, wantValue: math.MaxUint32 - 1},
	}

	Init()
	for i, intTest := range banScoreIntTests {
		rst := intTest.bs.int(time.Unix(intTest.timeLapse, 0))
		if rst != intTest.wantValue {
			t.Fatal("test ban score int err.", "num:", i, "want:", intTest.wantValue, "got:", rst)
		}
	}
}

func TestIncrease(t *testing.T) {
	var banScoreIncreaseTests = []struct {
		bs            DynamicBanScore
		transientAdd  uint32
		persistentAdd uint32
		timeLapse     int64
		wantValue     uint32
	}{
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, transientAdd: 50, persistentAdd: 50, timeLapse: 1, wantValue: 199},
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, transientAdd: 50, persistentAdd: 50, timeLapse: Lifetime, wantValue: 150},
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, transientAdd: 50, persistentAdd: 50, timeLapse: Lifetime + 1, wantValue: 150},
		{bs: DynamicBanScore{lastUnix: 0, transient: 50, persistent: 50}, transientAdd: 50, persistentAdd: 50, timeLapse: -1, wantValue: 200},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: 0}, transientAdd: math.MaxUint32, persistentAdd: 0, timeLapse: 60, wantValue: math.MaxUint32},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: 0}, transientAdd: 0, persistentAdd: math.MaxUint32, timeLapse: 60, wantValue: math.MaxUint32},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: 0}, transientAdd: 0, persistentAdd: math.MaxUint32, timeLapse: Lifetime + 1, wantValue: math.MaxUint32},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: 0}, transientAdd: math.MaxUint32, persistentAdd: 0, timeLapse: Lifetime + 1, wantValue: math.MaxUint32},
		{bs: DynamicBanScore{lastUnix: 0, transient: math.MaxUint32, persistent: 0}, transientAdd: math.MaxUint32, persistentAdd: 0, timeLapse: Lifetime + 1, wantValue: math.MaxUint32},
		{bs: DynamicBanScore{lastUnix: 0, transient: math.MaxUint32, persistent: 0}, transientAdd: math.MaxUint32, persistentAdd: 0, timeLapse: 0, wantValue: math.MaxUint32 - 1},
		{bs: DynamicBanScore{lastUnix: 0, transient: 0, persistent: math.MaxUint32}, transientAdd: math.MaxUint32, persistentAdd: 0, timeLapse: Lifetime + 1, wantValue: math.MaxUint32 - 1},
	}

	Init()
	for i, incTest := range banScoreIncreaseTests {
		rst := incTest.bs.increase(incTest.persistentAdd, incTest.transientAdd, time.Unix(incTest.timeLapse, 0))
		if rst != incTest.wantValue {
			t.Fatal("test ban score int err.", "num:", i, "want:", incTest.wantValue, "got:", rst)
		}
	}
}

func TestReset(t *testing.T) {
	var bs DynamicBanScore
	if bs.Int() != 0 {
		t.Errorf("Initial state is not zero.")
	}
	bs.Increase(100, 0)
	r := bs.Int()
	if r != 100 {
		t.Errorf("Unexpected result %d after ban score increase.", r)
	}
	bs.Reset()
	if bs.Int() != 0 {
		t.Errorf("Failed to reset ban score.")
	}
}

func TestString(t *testing.T) {
	want := "persistent 100 + transient 0 at 0 = 100 as of now"
	var bs DynamicBanScore
	if bs.Int() != 0 {
		t.Errorf("Initial state is not zero.")
	}

	bs.Increase(100, 0)
	if bs.String() != want {
		t.Fatal("DynamicBanScore String test error.")
	}
}
