package clock_test

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
)

// ENG-08: the drift signal compares the PRE-update local l with the wall
// clock. It therefore (a) fires after any idle period longer than driftWarn
// although every clock is perfectly synchronized, and (b) stays silent on the
// very event that absorbs a far-future remote version.
func TestReview_driftSignalSemantics(t *testing.T) {
	const warn = 60 * time.Second
	wall := time.UnixMilli(1_700_000_000_000)
	clk := clock.NewClock(func() time.Time { return wall }, 0, 0, warn)
	clk.Tick() // a local write at t0

	// (a) 61 s of idleness, then a record from a peer whose clock equals ours.
	wall = wall.Add(61 * time.Second)
	_, _, drift := clk.Update(wall.UnixMilli(), 0)
	t.Logf("(a) synced peer after 61 s idle: drift=%v", drift)
	if drift != 0 {
		t.Errorf("(a) false positive: drift=%v for a remote version equal to the local wall clock", drift)
	}

	// (b) a record from a peer whose clock is one hour ahead.
	_, _, drift = clk.Update(wall.Add(time.Hour).UnixMilli(), 0)
	t.Logf("(b) absorbing a version 1 h ahead of the wall clock: drift=%v", drift)
	if drift <= warn {
		t.Errorf("(b) missed: the received version runs 1h ahead of system time, drift=%v", drift)
	}

	// (c) from now on EVERY event reports the +1h drift, including local Ticks
	// and merges of perfectly sane remote versions.
	_, _, drift = clk.Update(wall.UnixMilli(), 0)
	t.Logf("(c) next synced remote version: drift=%v (counted as one more 'remote timestamp beyond the threshold')", drift)
}
