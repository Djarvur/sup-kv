package app_test

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app"
)

// expectedDrift is the documented drift convention as a model: evaluated on
// the pre-event state, the exact signed divergence when l and the wall clock
// are apart beyond warn in either direction, zero within tolerance (positive
// when l runs ahead, negative on a forward wall-clock jump).
func expectedDrift(lBefore, ptBefore int64, warn time.Duration) time.Duration {
	dMs := (lBefore - ptBefore) * int64(time.Millisecond)
	if dMs > int64(warn) || dMs < -int64(warn) {
		return time.Duration(dMs)
	}
	return 0
}

// TestClock_monotonic drives a 2000-step adversarial walk (backward and
// forward jumps, stuck clocks) where every step is a random Tick or an
// Update with a remote reading near the current state: every returned
// (l, counter) is strictly greater than the previous one, and the drift
// signal matches the documented sign convention at every step. A concurrent
// subtest hammers one clock from parallel Tick/Update callers and checks
// every caller's own readings stay monotonic with zero race-detector
// findings (VERS-03, TEST-02).
func TestClock_monotonic(t *testing.T) {
	t.Parallel()
	t.Run("sequential adversarial walk", func(t *testing.T) {
		t.Parallel()
		rng := newRNG(t)
		nowMs := testNowMs
		now := func() time.Time { return time.UnixMilli(nowMs) }
		clock := app.NewClock(now, 0, 0, testDriftWarn)
		prev := reading{}
		havePrev := false
		for i := range 2000 {
			nowMs = nextWallTime(rng, nowMs)
			lBefore, ptBefore := prev.l, nowMs
			got := reading{}
			var drift time.Duration
			if rng.IntN(2) == 0 {
				got.l, got.counter, drift = clock.Tick()
			} else {
				remoteL := max(0, prev.l+rng.Int64N(5)-2)
				got.l, got.counter, drift = clock.Update(remoteL, uint16(rng.IntN(testCounterMax+1)))
			}
			if havePrev && !got.after(prev) {
				t.Fatalf("step %d = (%d, %d), want strictly after (%d, %d) (replay with the property seed above)",
					i+1, got.l, got.counter, prev.l, prev.counter)
			}
			if want := expectedDrift(lBefore, ptBefore, testDriftWarn); drift != want {
				t.Fatalf("step %d drift = %v, want %v for pre-state (l=%d, pt=%d) (replay with the property seed above)",
					i+1, drift, want, lBefore, ptBefore)
			}
			prev = got
			havePrev = true
		}
	})
	t.Run("concurrent tick and update callers stay monotonic and race-free", func(t *testing.T) {
		t.Parallel()
		rng := newRNG(t)
		seedHi, seedLo := rng.Uint64(), rng.Uint64()
		var nowMs atomic.Int64
		nowMs.Store(testNowMs)
		now := func() time.Time { return time.UnixMilli(nowMs.Load()) }
		clock := app.NewClock(now, 0, 0, testDriftWarn)
		var wg sync.WaitGroup
		for worker := range 8 {
			wg.Go(func() {
				workerRng := rand.New(rand.NewPCG(seedHi, seedLo+uint64(worker)))
				prev := reading{}
				havePrev := false
				for range 500 {
					if workerRng.IntN(4) == 0 {
						nowMs.Store(nextWallTime(workerRng, nowMs.Load()))
					}
					got := reading{}
					if workerRng.IntN(2) == 0 {
						got.l, got.counter, _ = clock.Tick()
					} else {
						remoteL := max(0, nowMs.Load()+workerRng.Int64N(5)-2)
						got.l, got.counter, _ = clock.Update(remoteL, uint16(workerRng.IntN(testCounterMax+1)))
					}
					if havePrev && !got.after(prev) {
						t.Errorf("worker %d reading (%d, %d), want strictly after (%d, %d) (replay with the property seed above)",
							worker, got.l, got.counter, prev.l, prev.counter)
						return
					}
					prev = got
					havePrev = true
				}
			})
		}
		wg.Wait()
	})
}

// TestVersion_uniquePerWrite is the Pitfall-2 stress: 10,000 consecutive
// Ticks against a stuck clock, packed with one nodeID, are pairwise distinct
// and the whole packed sequence is strictly increasing — the 12-bit counter
// overflows into an l bump and never wraps or repeats (VERS-02).
func TestVersion_uniquePerWrite(t *testing.T) {
	t.Parallel()
	clock := app.NewClock(fixedNow(testNowMs), 0, 0, testDriftWarn)
	const writes = 10000
	seen := make(map[app.RecordVersion]bool, writes)
	var prev app.RecordVersion
	for i := range writes {
		l, counter, _ := clock.Tick()
		ver := app.PackVersion(l, counter, testNodeID)
		if seen[ver] {
			t.Fatalf("tick %d repeated version %x", i+1, ver[:])
		}
		seen[ver] = true
		if i > 0 && prev.Compare(ver) >= 0 {
			t.Fatalf("tick %d version %x, want strictly greater than %x", i+1, ver[:], prev[:])
		}
		prev = ver
	}
}

// TestClock_postReceive covers post-receive ordering (TEST-02): after an
// Update — or a full Merge into a populated store — the next local write
// packs a version strictly greater than the received one, so a local write
// after receiving always supersedes.
func TestClock_postReceive(t *testing.T) {
	t.Parallel()
	t.Run("the next tick after Update beats the remote version", func(t *testing.T) {
		t.Parallel()
		rng := newRNG(t)
		nowMs := testNowMs
		now := func() time.Time { return time.UnixMilli(nowMs) }
		clock := app.NewClock(now, 0, 0, testDriftWarn)
		lastL := int64(0)
		for i := range 1000 {
			nowMs = nextWallTime(rng, nowMs)
			var remoteL int64
			var remoteC uint16
			var remote app.RecordVersion
			if rng.IntN(2) == 0 {
				remoteL = max(0, max(lastL, nowMs)+rng.Int64N(3)-1)
				remoteC = uint16(rng.IntN(testCounterMax + 1))
				remote = app.PackVersion(remoteL, remoteC, uint16(rng.IntN(maxNodeID+1)))
			} else {
				remote = randRecord(rng).Version
				remoteL, remoteC, _ = remote.Unpack()
			}
			clock.Update(remoteL, remoteC)
			l, counter, _ := clock.Tick()
			local := app.PackVersion(l, counter, testNodeID)
			if local.Compare(remote) <= 0 {
				t.Fatalf("step %d local %x, want strictly greater than remote %x (replay with the property seed above)",
					i+1, local[:], remote[:])
			}
			lastL = l
		}
	})
	t.Run("a local write after Merge into a populated store supersedes the received version", func(t *testing.T) {
		t.Parallel()
		rng := newRNG(t)
		for i := range 100 {
			store := randStore(rng)
			key := randKey(rng)
			rec := randRecord(rng)
			_, _, err := store.Merge(key, rec)
			if err != nil {
				t.Fatalf("step %d Merge() error = %v, want nil", i, err)
			}
			ver, _, err := store.Set(key, []byte("local"))
			if err != nil {
				t.Fatalf("step %d Set() error = %v, want nil", i, err)
			}
			if ver.Compare(rec.Version) <= 0 {
				t.Fatalf("step %d local write %x, want strictly greater than received %x (replay with the property seed above)",
					i, ver[:], rec.Version[:])
			}
		}
	})
}

// TestClock_counterReset covers the counter-reset rule on a random walk
// whose wall clock advances beyond l: whenever l jumps to a new wall value
// the counter resets to 0 on that Tick, and otherwise the reading strictly
// increases (TEST-02).
func TestClock_counterReset(t *testing.T) {
	t.Parallel()
	rng := newRNG(t)
	nowMs := testNowMs
	now := func() time.Time { return time.UnixMilli(nowMs) }
	clock := app.NewClock(now, 0, 0, testDriftWarn)
	prev := reading{}
	for i := range 2000 {
		nowMs += rng.Int64N(3)
		l, counter, _ := clock.Tick()
		got := reading{l: l, counter: counter}
		var want reading
		switch {
		case nowMs > prev.l:
			want = reading{l: nowMs, counter: 0}
		case prev.counter == testCounterMax:
			want = reading{l: prev.l + 1, counter: 0}
		default:
			want = reading{l: prev.l, counter: prev.counter + 1}
		}
		if got != want {
			t.Fatalf("step %d at wall %d: tick = (%d, %d), want (%d, %d) (replay with the property seed above)",
				i+1, nowMs, got.l, got.counter, want.l, want.counter)
		}
		if !got.after(prev) {
			t.Fatalf("step %d tick = (%d, %d), want strictly after (%d, %d)",
				i+1, got.l, got.counter, prev.l, prev.counter)
		}
		prev = got
	}
}

// TestClock_restartProperty covers restart-from-dump-max under chaos
// (VERS-04, TEST-02): the dump models the FULL watermark — its watermark
// record is (dumpMax, watermarkC, testNodeID) with watermarkC drawn from
// 1..4095, counter >= 1 at the watermark millisecond carrying the store's
// own NodeID, the exact WR-01 collision hazard — plus eight older-l records
// with arbitrary counters and nodeIDs. A clock seeded with that full
// watermark (NewClock's initialC), under a wall clock jumped backward
// beyond dumpMax and a continued adversarial walk, keeps 100 Ticks at or
// above dumpMax, strictly increasing, with every packed emission beating
// every dump version including the watermark. This supersedes the
// counter-0 narrowing of the dump watermark recorded in the 02-02 summary,
// which 02-VERIFICATION falsified.
func TestClock_restartProperty(t *testing.T) {
	t.Parallel()
	rng := newRNG(t)
	dumpMax := testNowMs + rng.Int64N(1_000_000)
	watermarkC := uint16(1 + rng.IntN(testCounterMax))
	nowMs := dumpMax - 1 - rng.Int64N(4*60*60*1000)
	now := func() time.Time { return time.UnixMilli(nowMs) }
	clock := app.NewClock(now, dumpMax, watermarkC, testDriftWarn)
	dump := make([]app.RecordVersion, 0, 9)
	for range 8 {
		olderL := dumpMax - 1 - rng.Int64N(1_000_000)
		dump = append(dump, app.PackVersion(olderL, uint16(rng.IntN(testCounterMax+1)), uint16(rng.IntN(maxNodeID+1))))
	}
	dump = append(dump, app.PackVersion(dumpMax, watermarkC, testNodeID))
	prev := reading{l: dumpMax, counter: watermarkC}
	for i := range 100 {
		nowMs = nextWallTime(rng, nowMs)
		l, counter, _ := clock.Tick()
		got := reading{l: l, counter: counter}
		if got.l < dumpMax {
			t.Fatalf("tick %d l = %d, want at or above the dump max %d (replay with the property seed above)",
				i+1, got.l, dumpMax)
		}
		if !got.after(prev) {
			t.Fatalf("tick %d = (%d, %d), want strictly after (%d, %d)",
				i+1, got.l, got.counter, prev.l, prev.counter)
		}
		ver := app.PackVersion(l, counter, testNodeID)
		for _, dumpVer := range dump {
			if ver.Compare(dumpVer) <= 0 {
				t.Fatalf("tick %d version %x, want strictly greater than dump version %x (replay with the property seed above)",
					i+1, ver[:], dumpVer[:])
			}
		}
		prev = got
	}
}

// TestClock_driftUnderChaos proves both drift signs are observable on a long
// adversarial walk with DriftWarn 60s, and that the signal is exactly zero
// whenever the pre-event state is within DriftWarn of the wall clock with no
// jump beyond it in either direction (VERS-05, TEST-02).
func TestClock_driftUnderChaos(t *testing.T) {
	t.Parallel()
	rng := newRNG(t)
	nowMs := testNowMs
	now := func() time.Time { return time.UnixMilli(nowMs) }
	clock := app.NewClock(now, 0, 0, testDriftWarn)
	observedPositive := false
	observedNegative := false
	lastL := int64(0)
	for i := range 5000 {
		nowMs = nextWallTime(rng, nowMs)
		lBefore, ptBefore := lastL, nowMs
		got := reading{}
		var drift time.Duration
		if rng.IntN(2) == 0 {
			got.l, got.counter, drift = clock.Tick()
		} else {
			remoteL := max(0, lastL+rng.Int64N(5)-2)
			got.l, got.counter, drift = clock.Update(remoteL, uint16(rng.IntN(testCounterMax+1)))
		}
		if want := expectedDrift(lBefore, ptBefore, testDriftWarn); drift != want {
			t.Fatalf("step %d drift = %v, want %v for pre-state (l=%d, pt=%d) (replay with the property seed above)",
				i+1, drift, want, lBefore, ptBefore)
		}
		observedPositive = observedPositive || drift > 0
		observedNegative = observedNegative || drift < 0
		lastL = got.l
	}
	if !observedPositive || !observedNegative {
		t.Errorf("both drift signs must fire across the walk: positive=%t negative=%t",
			observedPositive, observedNegative)
	}
}
